<?php
$domain = $_SERVER['HTTP_HOST'];
$directory = $_SERVER['REQUEST_URI'];
$redirectUrl = 'https://' . $domain . '/#url=https://' . $domain . $directory;
header('Location: ' . $redirectUrl, true, 301);
exit;
?>