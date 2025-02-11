# Gopher2600-Utils

Simple utilities using the [Gopher2600](https://github.com/JetSetIlly/Gopher2600) engine. 

* audit
  	* Run audits on a single ROM or a collection of ROMs
  	* Audits are currently only writeable in Go and must be compiled into the executable
  	* Currently defined 'auditors' are:
  	  	* Frames generated with VSYNC but not VBLANK
  	  	* Screens drawn with hues 14 or 15
  	  	* Count the number each hue is used
  	* A significant limitation is that cartridges run from initialisation without user input
  	  	* This is a definite area of improvement for the future
  	 
* tiaAudio
	* Experimental WASM binary that allows TIA audio playback via JSON instruction
   	* Currently assumes a standard NTSC frame with audio update once per frame
   	* Example index.html demonstrates sample playback and updating via JSON
	
* ebiten_test
	* Demonstration of running the Gopher2600 engine with a small frontend
	* Also useful to show performance difference between native and WASM targets
	* webserve.sh will build a WASM binary and run a simple http daemon
	* No controllers attached to the emulation for now
	
* Example-Audit
	* Recursively scans a directory for 2600 cartridges
   	* Determines the bankswitching method for the ROM
	* Detects whether the TV output is NTSC or PAL
	* Tests whether any audio is output in first 60 frames
	
