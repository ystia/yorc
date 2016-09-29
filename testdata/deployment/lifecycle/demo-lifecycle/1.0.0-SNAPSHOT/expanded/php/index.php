<?php
header('Content-Type: text/html');
include 'common.php';
?>
<html>
<head>
<style>
table, th, td {
    border: 2px solid gray;
    border-collapse: collapse;
}
</style>
</head>
<body>
<?php
echo "<p><a href=\"log.php\">all debug logs</a></p>\n";

if (($handle = fopen("$registryFile", "r")) !== FALSE) {
    echo "<ul>\n";
    while (($data = fgetcsv($handle, 1000, ";")) !== FALSE) {
        $nodeName = $data[0];
        $instanceId = $data[1];
        $instanceIdx = $data[2];
        echo "<li>" . $nodeName . "[" . $instanceIdx . "] = " . $instanceId . " (<a href=\"get_instance_log.php?node=$nodeName&idx=$instanceIdx\">logs</a>)</li>\n";
    }
    fclose($handle);
    echo "</ul>\n";
}

if (($handle = fopen("$allLogFile", "r")) !== FALSE) {
    echo "<table border='1' style='width:100%'>\n";
    echo "<tr><th>#</th><th>Date</th><th>Node</th><th>Instance idx</th><th>Instance ID</th><th>Op.</th><th>Tiers Node</th><th>Tiers Instance idx</th><th>Tiers Instance ID</th><th></th></tr>\n";
    while (($data = fgetcsv($handle, 1000, ";")) !== FALSE) {
        $num = count($data);
        $operationIdx = $data[0];
        echo "<tr>\n";
        echo "<td>$operationIdx</td>\n";
        for ($c=1; $c < $num; $c++) {
            echo "<td>" . $data[$c] . "</td>\n";
        }
        echo "<td><a href=\"get_env_log.php?idx=$operationIdx\">env logs</a></td>";
        echo "</tr>\n";
    }
    fclose($handle);
    echo "</table>\n";
}
?>
</body>
</html>
