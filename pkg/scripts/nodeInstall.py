# XBD team
print "######### SSH Slurm Client ##########"
import sys

repoBase = "./"
sys.path.insert(0, repoBase)

from shRunner import ShRunnerAssync

if len(sys.argv) < 4:
    print "Usage : python nodeInstall.py user urlmo0 gpu"
    sys.exit(0)
else:
    USER = sys.argv[1]
    URL = sys.argv[2]
    GPU = sys.argv[3]
    print "Param : " + " user = " + USER + " url = " + URL + " gpu = " + GPU

GETJOBID = "cat /tmp/out \""

if GPU == "none":
    ALLOC = "salloc --no-shell -J xBD > /tmp/out 2>&1 \""
else:
    ALLOC = "salloc -p compute-rhel7.1 --gres=gpu --no-shell -J xBD > /tmp/out 2>&1 \""

co = "ssh  -o StrictHostKeyChecking=no " + USER + "@" + URL + " \" "

print ALLOC
ShRunnerAssync.runa(co + ALLOC)

print GETJOBID
jobeID = ShRunnerAssync.runaGet1rstLineOut(co + GETJOBID)
jobeID = jobeID.replace("\n", "")
tab = jobeID.split()
JOBID = tab[len(tab) - 1]
print "[runaPrintOut] " + JOBID

GETNODEID = "squeue -n xBD --noheader | grep " + JOBID + " | awk '{print \$(NF),\"\\t\"}' \""
print GETNODEID
nodeID = ShRunnerAssync.runaGet1rstLineOut(co + GETNODEID)
nodeID = nodeID.replace("\n", "")
print "[runaPrintOut] " + nodeID
