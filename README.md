# Gopher2600-Utils

Simple utilities using the [Gopher2600](https://github.com/JetSetIlly/Gopher2600) engine. 
	
* Ebiten Test
	* Demonstration of running the Gopher2600 engine with a small frontend
	* Also useful to show performance difference between native and WASM targets
	* webserve.sh will build a WASM binary and run a simple http daemon
	* No controllers attached to the emulation for now
	
* Example-Audit
	* Recursively scans a directory for 2600 cartridges
   	* Determines the bankswitching method for the ROM
	* Detects whether the TV output is NTSC or PAL
	* Tests whether any audio is output in first 60 frames
	
