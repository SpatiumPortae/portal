# This is a VHS (https://github.com/charmbracelet/vhs) tape file.

# ================ SETTINGS ================

Output ../demo-output/portal-sender-demo.gif

Set Shell bash
Set Theme "OneDark"
Set FontSize 18
Set Width 1280
Set Height 720

# ================ Require programs on PATH ================

Require portal

# ================ OUTPUT SOME INFO ================

Set TypingSpeed 35ms
Type "Let's send some files with Portal 🌌 ✨"
Sleep 2s
Set TypingSpeed 20ms
Backspace 36

# Hide emoji leftover garbage
Hide
Backspace 10
Type "clear"
Enter
Show

Set TypingSpeed 35ms
Type "ls"
Enter
Sleep 2s

Hide
Backspace 10
Type "clear"
Enter
Show

# Reset to human typing speed
Set TypingSpeed 50ms

# ================ START SENDING ================

Type "portal send"
Sleep 500ms
Type " portal-presentation.pdf"
Sleep 400ms
Type " cute-cat-pics"
Sleep 400ms
Type " secret-nuclear-codes.txt" 
Sleep 400ms
Type " fragmovie.mp4"
Sleep 2.5s
Enter 2

# wait for compression, should take ~3s, add some extra margin
Sleep 5s

# ================ SHOW OFF FILE LIST NAVIGATION ================

Set TypingSpeed 600ms
Down 3
Sleep 2s
Set TypingSpeed 400ms
Up 3
Sleep 2s

# Reset to human typing speed
Set TypingSpeed 50ms

# ================ SHOW OFF CLIPBOARD COPY ================

# press copy-to-clipboard key
Type "c"
Sleep 14s

# ================ TRANSFER SHOULD START ================

# Wait for transfer to finish + some extra time
Sleep 40s


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
