How to setup and start chunkymonkey on an Android device.
=========

Introduction
-----

Chunkymonkey is written in Go, and it's easy to compile binaries to run in ARM
devices. So we can run a minecraft server on an android mobile phone, easily.  

Note that this is just for fun. We don't expect that anybody will want to run a
serious minecraft server in a mobile device. Performance in a Nexus One is
really bad and there are no explicit plans to optimize chunkymonkey for this
use case.

![Chunkymonkey on Android][1]

Prerequisites
-------------
*   Android device with USB debugging enabled
*   adb installed on your computer (Android SDK)
*   Android connected to the same wireless network of your client, so you can connect to <android IP>:<port 25565>.

Initial setup
-------------

First, compile Go for ARM.

	export GOARCH=arm
	cd $GOROOT/src
	./all.bash

Compile an ARM binary for chunkymonkey:

	cd ~/src/chunkymonkey
	make clean ; make server

Ensure your android device is connected via USB and detected by adb.

	adb devices


Copy your minecraft map to your android device.

	adb push ~/.minecraft/saves/Tonga\ da\ Mironga/ /data/local/minecraft-save

Copy the chunkymonkey data files:

	for f in *json;do adb push $f /data/local/bin;done

Run chunkymonkey:

	adb shell "cd /data/local/bin ; chmod 700 chunkymonkey; ./chunkymonkey /data/local/minecraft-save/"

[1]: ../../raw/master/docs/android.png  "Chunkymonkey on Android"
