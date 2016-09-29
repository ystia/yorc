<?php
header('Content-Type: text/plain');
include 'common.php';

$nodeName = $_GET["node"];
$instanceId = $_GET["instance"];
$tierNodeName = $_GET["tierNode"];
$tierInstanceId = $_GET["tierInstance"];
$operation = $_GET["operation"];
$envLog = file_get_contents('php://input');

$nodeLogPath = "$logPath/$nodeName";
if (!file_exists($nodeLogPath)) {
  mkdir($nodeLogPath, 0700, true);
}

$instanceIdx = getIntanceIdx($nodeName, $instanceId, $registryPath);
$tierInstanceIdx = getIntanceIdx($tierNodeName, $tierInstanceId, $registryPath);

$operationIdx = getOperationIdx($opLogPath);
file_put_contents("$opLogPath/$operationIdx.env", "$envLog");

$instanceLogFile = "$nodeLogPath/$instanceIdx";
file_put_contents($instanceLogFile, "#$operationIdx - $operation/$tierNodeName/$tierInstanceIdx/$tierInstanceId\n", FILE_APPEND | LOCK_EX);
file_put_contents("$allLogFile", "$operationIdx;" . date("Y-m-d H:i:s").";$nodeName;$instanceIdx;$instanceId;$operation;$tierNodeName;$tierInstanceIdx;$tierInstanceId\n", FILE_APPEND | LOCK_EX);
?>
