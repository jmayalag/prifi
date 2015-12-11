#!/bin/bash

echo "Pinging the control machine..."
ping -c 20 192.168.253.1

echo "Pinging the client 0"
ping -c 20 10.0.0.1

echo "Pinging the client 1"
ping -c 20 10.0.0.2

echo "Pinging the client 2"
ping -c 20 10.0.0.3

echo "Pinging the client 3"
ping -c 20 10.0.0.4

echo "Pinging the client 4"
ping -c 20 10.0.0.5

echo "Pinging the client 5"
ping -c 20 10.0.0.6

echo "Pinging the client 6"
ping -c 20 10.0.0.7

echo "Pinging the client 7"
ping -c 20 10.0.0.8

echo "Pinging the client 8"
ping -c 20 10.0.0.9

echo "Pinging the client 9"
ping -c 20 10.0.0.10

echo "Pinging the trustee 0"
ping -c 20 10.0.1.1

echo "Pinging the trustee 1"
ping -c 20 10.0.1.2

echo "Pinging the trustee 2"
ping -c 20 10.0.1.3

echo "Pinging the trustee 3"
ping -c 20 10.0.1.4

echo "Pinging the trustee 4"
ping -c 20 10.0.1.5
