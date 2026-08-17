Output ../public/recordings/ci-smoke-test.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 800
Set Padding 20
Set Theme "Dracula"

Type "cd $(git rev-parse --show-toplevel)"
Enter

Type "./bin/maelsink serve --headless --db=./ci-smoke.db &"
Enter
Sleep 1500ms

Type "clear"
Enter

Type "curl -sf http://127.0.0.1:9090/api/v1/health && echo ' -> ready'"
Enter
Sleep 1000ms

Type "./bin/maelsink send --to=ci@example.com --subject='Signup confirmation' --text='Please confirm your account'"
Enter
Sleep 800ms

Type "curl -s 'http://127.0.0.1:9090/api/v1/messages?subject=Signup' | head -c 400"
Enter
Sleep 1500ms

Type "kill %1 && rm -f ./ci-smoke.db"
Enter
Sleep 500ms
