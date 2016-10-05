<?php
header('Content-Type: text/plain');
include 'common.php';

$operationIdx = $_GET["idx"];
$opFilePath = "$opLogPath/$operationIdx.env";
if (file_exists($opFilePath)) {
  echo file_get_contents($opFilePath);
}
?>
