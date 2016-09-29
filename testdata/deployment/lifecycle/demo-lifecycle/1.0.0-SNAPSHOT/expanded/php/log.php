<?php
header('Content-Type: text/plain');
include 'common.php';

echo file_get_contents("$logFile");
?>