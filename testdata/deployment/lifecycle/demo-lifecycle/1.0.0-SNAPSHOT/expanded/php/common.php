<?php
$logPath = "/tmp/a4c_log";
if (!file_exists($logPath)) {
  mkdir($logPath, 0700, true);
}
$opLogPath = "/tmp/a4c_oplog";
if (!file_exists($opLogPath)) {
  mkdir($opLogPath, 0700, true);
}

$allLogFile = "$logPath/all";

$logFile = "$logPath/log.txt";
file_put_contents($logFile, date("Y-m-d H:i:s") . " : " . $_SERVER['PHP_SELF'] . " - called\n", FILE_APPEND | LOCK_EX);
foreach ($_GET as $name => $value) {
    file_put_contents($logFile, date("Y-m-d H:i:s") . " : " . $_SERVER['PHP_SELF'] . " - $name=$value\n", FILE_APPEND | LOCK_EX);
}

$registryPath = "/tmp/a4c_registry";
$registryFile = "$registryPath/registryFile";

function getOperationIdx($opLogPath) {
  $operationIdxLockFile = "$opLogPath/opLock";
  while (true) {
    if (file_exists($operationIdxLockFile)) {
      sleep(1);
    } else {
      touch($operationIdxLockFile);
      $operationIdxFile = "$opLogPath/opIdx";
      if (file_exists($operationIdxFile)) {
        $opIdx = intval(file_get_contents($operationIdxFile));
        $nextOpIdx = $opIdx + 1;
        file_put_contents($operationIdxFile, "$nextOpIdx");
      } else {
        $opIdx = 0;
        file_put_contents($operationIdxFile, "1");
      }
      unlink($operationIdxLockFile);
      break;
    }
  }
  return $opIdx;
}

function getIntanceIdx($nodeName, $instanceId, $registryPath)
{
  $registryFile = "$registryPath/registryFile";
  $registryNodePath = "$registryPath/$nodeName";
  $instanceFile = "$registryNodePath/$instanceId";

  if (file_exists($instanceFile)) {
    $instanceIdx = intval(file_get_contents($instanceFile));
  } else {

    if (!file_exists($registryNodePath)) {
      mkdir($registryNodePath, 0700, true);
    }
    $indexFile = "$registryNodePath/index";
    $lockFile = "$registryNodePath/lock";

    while (true) {
      if (file_exists($lockFile)) {
        sleep(1);
      } else {
        touch($lockFile);
        if (file_exists($indexFile)) {
          $instanceIdx = intval(file_get_contents($indexFile));
          $nextInstanceIdx = $instanceIdx + 1;
          file_put_contents($indexFile, "$nextInstanceIdx");
        } else {
          $instanceIdx = 0;
          file_put_contents($indexFile, "1");
        }
        unlink($lockFile);
        file_put_contents($registryFile, "$nodeName;$instanceId;$instanceIdx\n", FILE_APPEND | LOCK_EX);
        file_put_contents($instanceFile, "$instanceIdx");
        break;
      }
    }
  }
  return $instanceIdx;
}

?>
