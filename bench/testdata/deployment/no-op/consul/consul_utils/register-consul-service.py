#!/usr/bin/env python
from __future__ import print_function

import argparse
import httplib
import json
import sys

import contextlib

__author__ = 'Loic Albertin'


def port_range(string):
    value = int(string)
    if value in range(1, 65535 + 1):
        return value
    msg = "%r is not in range [1 ... 65535]" % string
    raise argparse.ArgumentTypeError(msg)


def main():
    parser = argparse.ArgumentParser(description='Generate a Consul Service configuration file')
    parser.add_argument('-s', '--service', dest='service', action='store',
                        type=str, nargs=1, required=True,
                        help='The service name')
    parser.add_argument('-a', '--address', dest='address', action='store',
                        type=str, nargs=1, required=True,
                        help='The IP address that hosts the service')
    parser.add_argument('-p', '--port', dest='port', action='store',
                        type=port_range, nargs='?', default=None,
                        help='The port used to access the service')
    parser.add_argument('-t', '--tag', dest='tags', action='store',
                        type=str, nargs='*', default=None,
                        help='Optionals tags used to register and lookup this service')
    parser.add_argument('-ca', '--consul-address', dest='consul_address', action='store',
                        type=str, nargs=1, default='localhost',
                        help='Consul http API address')
    parser.add_argument('-cp', '--consul-port', dest='consul_port', action='store',
                        type=port_range, nargs=1, default=8500,
                        help='Consul http API port')
    parser.add_argument('-c', '--check', dest='checks', action='store',
                        type=str, nargs='*', default='',
                        help='Enable service check and define to which. The format of a check is "<type>,<interval>,<check> where type is one of..."')

    args = parser.parse_args()

    service = {'name': args.service[0], 'address': args.address[0]}
    if args.port:
        service['port'] = args.port
    if args.tags:
        service['tags'] = args.tags
    if args.checks:
        service_check = {"Interval": None}
        for check in args.checks:
            check_argument = check.split(",")

            check_type = check_argument[0]
            interval = check_argument[1]
            ref = check_argument[2]

            service_check[check_type] = ref
            # Only last interval parameter will be taken into account
            service_check["Interval"] = interval

        service["Check"] = service_check


    with contextlib.closing(httplib.HTTPConnection(args.consul_address, args.consul_port)) as conn:
        conn.request("PUT", "/v1/agent/service/register", json.dumps(service))
        response = conn.getresponse()
        if response.status != 200:
            print("Registering service failed: {code} {reason}".format(code=response.status,reason=response.reason), file=sys.stderr)
            sys.exit(response.status)
    print("Service {name} successfully registered to consul.".format(name=args.service[0]))

if __name__ == "__main__":
    main()
