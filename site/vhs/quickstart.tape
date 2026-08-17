Output ../public/recordings/quickstart.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 20
Set Theme "Dracula"

Type "cd $(git rev-parse --show-toplevel)"
Enter

Type "./bin/maelsink serve &"
Enter
Sleep 1500ms

Type "clear"
Enter

Type "./bin/maelsink send --to=dev@example.com --subject='Welcome' --text='Hello from maelsink'"
Enter
Sleep 800ms

Type "./bin/maelsink list"
Enter
Sleep 1500ms

Type "kill %1"
Enter
Sleep 500ms
