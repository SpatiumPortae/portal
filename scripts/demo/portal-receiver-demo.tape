# This is a VHS (https://github.com/charmbracelet/vhs) tape file.

# ================ SETTINGS ================

Output ../demo-output/portal-receiver-demo.gif

Set Shell bash
Set Theme "OneDark"
Set FontSize 18
Set Width 1280
Set Height 720

# ================ Require programs on PATH ================

Require portal

# ================ WAITING FOR PASSWORD ================

Set TypingSpeed 35ms
Type "Let's receive some files with Portal 🌌 ✨"
Sleep 2s
Set TypingSpeed 20ms
Backspace 39

# Hide emoji leftover garbage
Hide
Backspace 10
Type "clear"
Enter
Show

Set TypingSpeed 35ms
Type "We currently have no files on this end..."
Sleep 2s
Set TypingSpeed 20ms
Backspace 41

Set TypingSpeed 35ms
Type "ls"
Enter
Sleep 2s

Hide
Backspace 10
Type "clear"
Enter
Show

Set TypingSpeed 35ms
Type "We are waiting for the sender to tell us the password..."
Sleep 2s
Set TypingSpeed 20ms
Backspace 56

Set TypingSpeed 35ms
Type "This is a great time to star ⭐✨ Portal on GitHub 💕"
Sleep 3s
Set TypingSpeed 20ms
Backspace 49

# Hide emoji leftover garbage
Hide
Backspace 10
Type "clear"
Enter
Show

Set TypingSpeed 20ms
Type "Okay! The sender has communicated to us that the password is '2-universe-phase-beta', let's go!"
Set TypingSpeed 20ms
Backspace 95

# ================ START RECEIVING ================

# Reset to human typing speed
Set TypingSpeed 50ms

Hide
Type "portal receive 2-universe-phase-beta"
Show
Sleep 1s
Enter 2

# ~transfer time + extra to gaze in the glory of Portal
Sleep 30s

Type "ls"
Enter
Sleep 9s


# === DOCS for reference === 

# VHS documentation
#
# Output:
#   Output <path>.gif               Create a GIF output at the given <path>
#   Output <path>.mp4               Create an MP4 output at the given <path>
#   Output <path>.webm              Create a WebM output at the given <path>
#
# Require:
#   Require <string>                Ensure a program is on the $PATH to proceed
#
# Settings:
#   Set FontSize <number>           Set the font size of the terminal
#   Set FontFamily <string>         Set the font family of the terminal
#   Set Height <number>             Set the height of the terminal
#   Set Width <number>              Set the width of the terminal
#   Set LetterSpacing <float>       Set the font letter spacing (tracking)
#   Set LineHeight <float>          Set the font line height
#   Set LoopOffset <float>%         Set the starting frame offset for the GIF loop
#   Set Theme <json|string>         Set the theme of the terminal
#   Set Padding <number>            Set the padding of the terminal
#   Set Framerate <number>          Set the framerate of the recording
#   Set PlaybackSpeed <float>       Set the playback speed of the recording
#
# Sleep:
#   Sleep <time>                    Sleep for a set amount of <time> in seconds
#
# Type:
#   Type[@<time>] "<characters>"    Type <characters> into the terminal with a
#                                   <time> delay between each character
#
# Keys:
#   Backspace[@<time>] [number]     Press the Backspace key
#   Down[@<time>] [number]          Press the Down key
#   Enter[@<time>] [number]         Press the Enter key
#   Space[@<time>] [number]         Press the Space key
#   Tab[@<time>] [number]           Press the Tab key
#   Left[@<time>] [number]          Press the Left Arrow key
#   Right[@<time>] [number]         Press the Right Arrow key
#   Up[@<time>] [number]            Press the Up Arrow key
#   Down[@<time>] [number]          Press the Down Arrow key
#   PageUp[@<time>] [number]        Press the Page Up key
#   PageDown[@<time>] [number]      Press the Page Down key
#   Ctrl+<key>                      Press the Control key + <key> (e.g. Ctrl+C)
#
# Display:
#   Hide                            Hide the subsequent commands from the output
#   Show                            Show the subsequent commands in the output
