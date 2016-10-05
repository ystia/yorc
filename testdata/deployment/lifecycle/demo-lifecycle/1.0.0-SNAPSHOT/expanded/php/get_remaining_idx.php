<?php
header('Content-Type: text/plain');
include 'common.php';

$nodeName = $_GET["node"];
$positionStr = $_GET["position"];

# 0 means the last line, 1 the line before the last ...
$position = intval($positionStr);

$nodeLogPath = "$logPath/$nodeName";
$stoppedLogFile = "$nodeLogPath/stopped";

# feed and array containing all stopped indexes
$stoppedLogContent = file($stoppedLogFile);
$lineCount = count($stoppedLogContent);
$stoppedArray = array();
foreach ($stoppedLogContent as $lineNumber => $lineContent)
{
  $stoppedArray[$lineNumber] = str_replace("\n","",$lineContent);
}

# feed an array containing all possible indexes
$registryNodePath = "$registryPath/$nodeName";
$indexFile = "$registryNodePath/index";
$instanceCount = intval(file_get_contents($indexFile));
$possibleArray = array();
for ($i = 0; $i < $instanceCount; $i++) {
  $possibleArray[$i] = "$i";
}

# this is the array of remaining indexes
$result = array_values(array_diff($possibleArray, $stoppedArray));

# just return the requested one
echo $result[$position];
?>
