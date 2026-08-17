Output ../public/recordings/send-command.gif

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

Type "./bin/maelsink send --help"
Enter
Sleep 2000ms

Type "clear"
Enter

Type "./bin/maelsink send --to=alice@example.com --to=bob@example.com --subject='Report ready' --html='<b>See attached</b>' --attach=./go.mod"
Enter
Sleep 800ms

Type "./bin/maelsink list --to=alice"
Enter
Sleep 1500ms

Type "kill %1"
Enter
Sleep 500ms
