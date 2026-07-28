package archive

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	ContainerName = "webrecorder"
	ImageName     = "site-archiving-toolkit-webrecorder"
)

// DockerAvailable checks whether the Docker daemon is reachable.
func DockerAvailable() error {
	cmd := exec.Command("docker", "ps")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("docker is not available: %s", msg)
	}
	return nil
}

// IsCrawlRunning reports whether a webrecorder container is already running.
func IsCrawlRunning() (bool, error) {
	cmd := exec.Command("docker", "ps", "-q", "-f", "name="+ContainerName)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// BuildImage builds the webrecorder Docker image.
func BuildImage(ctx context.Context, rootDir string, logFn func(string)) error {
	dockerfile := filepath.Join("resources", "Dockerfile.webrecorder")
	cmd := exec.CommandContext(ctx, "docker", "build",
		"-f", dockerfile,
		".",
		"-t", ImageName,
	)
	cmd.Dir = rootDir
	return streamCmd(cmd, logFn)
}

// RunOptions configures a single crawl container run.
type RunOptions struct {
	RootDir       string
	CrawlDir      string
	URL           string
	NormalizedURL string
	Timestamp     string
	ArchiveINI    string // path to archive.ini on host
}

// CrawlProcess represents a running crawl container.
type CrawlProcess struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	done   chan error
	once   sync.Once
}

// StartCrawl starts the webrecorder container and streams logs via logFn.
// It returns a CrawlProcess that can be waited on or stopped.
func StartCrawl(ctx context.Context, opts RunOptions, logFn func(string)) (*CrawlProcess, error) {
	ctx, cancel := context.WithCancel(ctx)

	webrecorderDir := filepath.Join(opts.CrawlDir, "webrecorder")
	if err := os.MkdirAll(webrecorderDir, 0o777); err != nil {
		cancel()
		return nil, err
	}

	args := []string{
		"run",
		"--name", ContainerName,
		"--rm",
		"-v", opts.CrawlDir + ":/output",
		"-v", opts.ArchiveINI + ":/archive.ini:ro",
		ImageName,
		"bash", "/webrecorder.sh", opts.URL, opts.NormalizedURL, opts.Timestamp,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = opts.RootDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start container: %w", err)
	}

	cp := &CrawlProcess{
		cmd:    cmd,
		cancel: cancel,
		done:   make(chan error, 1),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		scanLines(stdout, logFn)
	}()
	go func() {
		defer wg.Done()
		scanLines(stderr, logFn)
	}()

	go func() {
		wg.Wait()
		err := cmd.Wait()
		cp.done <- err
		close(cp.done)
	}()

	return cp, nil
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

// QuitCrawlers stops any running webrecorder containers.
func QuitCrawlers() (string, error) {
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

// StartServer starts the local preview Apache server.
func StartServer(rootDir string) error {
	compose := filepath.Join(rootDir, "resources", "docker-compose.yml")
	cmd := exec.Command("docker", "compose", "-f", compose, "up", "-d")
	cmd.Dir = rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Fall back to docker-compose binary
		cmd = exec.Command("docker-compose", "-f", compose, "up", "-d")
		cmd.Dir = rootDir
		out2, err2 := cmd.CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("start server: %v (%s)", err, strings.TrimSpace(string(out)+" "+string(out2)))
		}
	}
	return nil
}

// StopServer stops the local preview Apache server.
func StopServer(rootDir string) error {
	compose := filepath.Join(rootDir, "resources", "docker-compose.yml")
	cmd := exec.Command("docker", "compose", "-f", compose, "down")
	cmd.Dir = rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		cmd = exec.Command("docker-compose", "-f", compose, "down")
		cmd.Dir = rootDir
		out2, err2 := cmd.CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("stop server: %v (%s)", err, strings.TrimSpace(string(out)+" "+string(out2)))
		}
	}
	return nil
}

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
	go func() {
		defer wg.Done()
		scanLines(stdout, logFn)
	}()
	go func() {
		defer wg.Done()
		scanLines(stderr, logFn)
	}()
	wg.Wait()
	return cmd.Wait()
}

func scanLines(r io.Reader, logFn func(string)) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		logFn(scanner.Text())
	}
}
