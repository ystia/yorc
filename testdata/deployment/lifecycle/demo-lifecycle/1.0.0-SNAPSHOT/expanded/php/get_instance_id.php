<?php
header('Content-Type: text/plain');
include 'common.php';

$nodeName = $_GET["node"];
$instanceIdx = $_GET["idx"];

if (($handle = fopen("$registryFile", "r")) !== FALSE) {
  while (($data = fgetcsv($handle, 1000, ";")) !== FALSE) {
    $thisNodeName = $data[0];
    $thisInstanceId = $data[1];
    $thisInstanceIdx = $data[2];
    if ($nodeName == $thisNodeName && $instanceIdx == $thisInstanceIdx) {
      echo $thisInstanceId;
    }
  }
  fclose($handle);
}

?>
