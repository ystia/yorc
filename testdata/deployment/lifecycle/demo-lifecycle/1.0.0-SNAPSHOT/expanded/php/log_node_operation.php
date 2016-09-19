<?php
header('Content-Type: text/plain');
include 'common.php';

$nodeName = $_GET["node"];
$instanceId = $_GET["instance"];
$operation = $_GET["operation"];
$envLog = file_get_contents('php://input');

$nodeLogPath = "$logPath/$nodeName";
if (!file_exists($nodeLogPath)) {
  mkdir($nodeLogPath, 0700, true);
}

$instanceIdx = getIntanceIdx($nodeName, $instanceId, $registryPath);
$operationIdx = getOperationIdx($opLogPath);
file_put_contents("$opLogPath/$operationIdx.env", "$envLog");

$logFile = "$nodeLogPath/$instanceIdx";
file_put_contents($logFile, "#$operationIdx - $operation\n", FILE_APPEND | LOCK_EX);
file_put_contents("$allLogFile", "$operationIdx;" . date("Y-m-d H:i:s").";$nodeName;$instanceIdx;$instanceId;$operation;;;\n", FILE_APPEND | LOCK_EX);

if ($operation == "stop") {
  $stoppedLogFile = "$nodeLogPath/stopped";
  file_put_contents("$stoppedLogFile", "$instanceIdx\n", FILE_APPEND | LOCK_EX);  
}
?>
