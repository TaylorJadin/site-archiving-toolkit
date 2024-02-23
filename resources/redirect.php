<?php
$redirectUrl = 'https://' . $_SERVER['HTTP_HOST'] . '/#url=' . 'CRAWL_URL' . $_SERVER['REQUEST_URI'];
header('Location: ' . $redirectUrl, true, 301);
exit;
?>