<?php
header('Content-Type: text/plain');
include 'common.php';

$nodeName = $_GET["node"];
$instanceIdx = $_GET["idx"];

$nodeLogPath = "$logPath/$nodeName";
$logFile = "$nodeLogPath/$instanceIdx";
echo file_get_contents($logFile);
?>