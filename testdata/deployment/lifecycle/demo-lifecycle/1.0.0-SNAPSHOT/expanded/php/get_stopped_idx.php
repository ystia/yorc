<?php
header('Content-Type: text/plain');
include 'common.php';

$nodeName = $_GET["node"];
$positionStr = $_GET["position"];

# 0 means the last line, 1 the line before the last ...
$position = intval($positionStr);

$nodeLogPath = "$logPath/$nodeName";
$stoppedLogFile = "$nodeLogPath/stopped";

$stoppedLogContent = file($stoppedLogFile);
$lineCount = count($stoppedLogContent);
$lineToFetchIdx = $lineCount - $position - 1;
$line = $stoppedLogContent[$lineToFetchIdx];
$line = str_replace("\n","",$line);
echo $line;

?>
