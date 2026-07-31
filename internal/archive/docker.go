package archive

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/TaylorJadin/site-archiving-toolkit/resources"
)

const (
	// ContainerName is the crawler container; only one may run at a time.
	ContainerName = "webrecorder"
	// PreviewContainerName serves crawls/ over HTTP on port 80.
	PreviewContainerName = "site-archiving-toolkit-preview"
	// ImageName is the locally built crawler image.
	ImageName = "site-archiving-toolkit-webrecorder"
)

// DockerAvailable checks whether the Docker daemon is reachable.
func DockerAvailable() error {
	out, err := exec.Command("docker", "ps").CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("docker is not available: %s", msg)
	}
	return nil
}

// containerRunning reports whether a container with exactly this name is up.
func containerRunning(name string) (bool, error) {
	out, err := exec.Command("docker", "ps", "-q", "-f", "name=^"+name+"$").Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// IsCrawlRunning reports whether a crawler container is already running.
func IsCrawlRunning() (bool, error) {
	return containerRunning(ContainerName)
}

// BuildImage builds the crawler image from the embedded build context.
func BuildImage(ctx context.Context, logFn func(string)) error {
	dir, err := os.MkdirTemp("", "site-archiving-toolkit-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	entries, err := fs.ReadDir(resources.BuildContext, ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := resources.BuildContext.ReadFile(e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o644); err != nil {
			return err
		}
	}

	// -f is resolved against the caller's working directory, not the context.
	cmd := exec.CommandContext(ctx, "docker", "build",
		"-f", filepath.Join(dir, resources.Dockerfile), "-t", ImageName, dir)
	return streamCmd(cmd, logFn)
}

// RunOptions configures a single crawl container run.
type RunOptions struct {
	CrawlDir      string
	URL           string
	NormalizedURL string
	Timestamp     string
	Env           []string
}

// CrawlProcess represents a running crawl container.
type CrawlProcess struct {
	cancel context.CancelFunc
	done   chan error
	once   sync.Once
}

// StartCrawl starts the crawler container and streams its output via logFn.
// The returned CrawlProcess can be waited on or stopped.
func StartCrawl(ctx context.Context, opts RunOptions, logFn func(string)) *CrawlProcess {
	ctx, cancel := context.WithCancel(ctx)

	args := []string{"run", "--name", ContainerName, "--rm", "-v", opts.CrawlDir + ":/output"}
	for _, e := range opts.Env {
		args = append(args, "-e", e)
	}
	args = append(args, ImageName, "bash", "/webrecorder.sh", opts.URL, opts.NormalizedURL, opts.Timestamp)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cp := &CrawlProcess{cancel: cancel, done: make(chan error, 1)}

	go func() {
		cp.done <- streamCmd(cmd, logFn)
		close(cp.done)
	}()

	return cp
}

// Wait blocks until the crawl finishes.
func (c *CrawlProcess) Wait() error {
	return <-c.done
}

// Stop stops the crawl container (skip or cancel).
func (c *CrawlProcess) Stop() {
	c.once.Do(func() {
		_ = exec.Command("docker", "stop", ContainerName).Run()
		c.cancel()
	})
}

// QuitCrawlers stops the current crawl. An active session runner is asked to
// cancel so it abandons the rest of its queue; a container left behind without
// a runner is stopped directly.
func QuitCrawlers(rootDir string) (string, error) {
	if ActiveSession(rootDir) != nil {
		if err := SendControl(rootDir, "cancel"); err != nil {
			return "", err
		}
		return "Stopping Browsertrix Crawler.", nil
	}
	running, err := IsCrawlRunning()
	if err != nil {
		return "", err
	}
	if !running {
		return "Browsertrix Crawler is not running.", nil
	}
	if err := exec.Command("docker", "stop", ContainerName).Run(); err != nil {
		return "", err
	}
	return "Successfully quit Browsertrix Crawler.", nil
}

// StartServer serves the crawls directory over HTTP on port 80.
func StartServer(rootDir string) error {
	crawls := filepath.Join(rootDir, "crawls")
	if err := os.MkdirAll(crawls, 0o777); err != nil {
		return err
	}
	_ = exec.Command("docker", "rm", "-f", PreviewContainerName).Run()
	out, err := exec.Command("docker", "run", "-d",
		"--name", PreviewContainerName,
		"--restart", "unless-stopped",
		"-p", "80:80",
		"-v", crawls+":/usr/local/apache2/htdocs",
		"httpd",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("start server: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// StopServer stops the local preview server.
func StopServer() error {
	out, err := exec.Command("docker", "rm", "-f", PreviewContainerName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop server: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// streamCmd runs cmd, forwarding each line of stdout and stderr to logFn.
func streamCmd(cmd *exec.Cmd, logFn func(string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(2)
	for _, r := range []io.Reader{stdout, stderr} {
		go func() {
			defer wg.Done()
			scanLines(r, logFn)
		}()
	}
	wg.Wait()
	return cmd.Wait()
}

func scanLines(r io.Reader, logFn func(string)) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		logFn(scanner.Text())
	}
}
