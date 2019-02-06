# RangeDown

RangeDown is a lightweight, multi-connection file download program written in Go.

## Work in Progress

The main idea is to have a program/library that does the following:

* Donwload a single file using multiple connections by checking if the server supports the Accept-Ranges header
* Have both a CLI and library that can be used by other programs/libraries
* Support downloading multiple files at once
* Provide rich information about progress of each individual download part.
* Support resuming for previous downloads.


This code is partially implemented and development should be resumed (once I have more time) 
