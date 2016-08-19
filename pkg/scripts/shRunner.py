import os
import random
import sys
from threading import Thread, RLock
import time
import subprocess


class ShRunnerAssync(Thread):
    def __init__(self, cmd, printASAP=False):
        super(ShRunnerAssync, self).__init__()
        self.cmd = cmd
        self.printASAP = printASAP

    def runsh(self, command):
        sh = subprocess.Popen([command], shell=True,
                              stdout=subprocess.PIPE,
                              stderr=subprocess.PIPE, bufsize=1)
        return sh

    def printRes(self):
        res = self.readResults()
        print "#### CDM : " + res[0]
        print len(res[1])
        print "#### OUT : " + str(res[1])
        print "#### ERR : " + str(res[2])
        print "	FIN ####"

    def run(self):
        self.process = self.runsh(self.cmd)
        if self.printASAP:
            self.printResASAP()

    def readResults(self):
        return (
            self.cmd,
            self.process.stdout.readlines(),
            self.process.stderr.readlines()
        )

    def printResASAP(self):
        self.process.wait()
        self.printRes()

    def getOut(self):
        return self.process.stdout.readlines()

    @classmethod
    def runa(cls, cmd):
        runner = ShRunnerAssync(cmd)
        runner.start()
        runner.join()
        runner.printRes()

    @classmethod
    def runaPrint1rstLineOut(cls, cmd):
        runner = ShRunnerAssync(cmd)
        runner.start()
        runner.join()
        res = runner.readResults()
        print "[runaPrintOut] " + str(res[1][0])

    @classmethod
    def runaGet1rstLineOut(cls, cmd):
        runner = ShRunnerAssync(cmd)
        runner.start()
        runner.join()
        res = runner.readResults()
        if (len(res) == 0):
            print "no tab result for " + cmd
            return "ERROR"
        else:
            if (len(res[1]) == 0):
                print "no stdout for " + cmd
                print str(res[2])
                return "ERROR"
        return str(res[1][0])

    @classmethod
    def runaNoPrint(cls, cmd):
        runner = ShRunnerAssync(cmd)
        runner.start()
        runner.join()
