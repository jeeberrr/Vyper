Sorry about the file naming conventions go doesent like it if i import two different things from different directories but for the //go:build i needed two things from the same package for linux and windows so i had to put them in the same dir. heres a general part of the naming convention for you

filename_windows.go (//go:build windows version)
filename_linux.go (//go:build linux version)
filename_mac.go (//go:build darwin version)
filename_unix.go (//go:build linux || darwin version) (linux/mac)
filename.go (global file EX: firefox.go)