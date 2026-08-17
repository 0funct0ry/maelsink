Output ../public/recordings/telnet-maelsink-session.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 700
Set Padding 20
Set Theme "Dracula"


Type "telnet localhost 1025"
Enter
Sleep 1s

Type "EHLO localhost"
Enter
Sleep 1s

Type "MAIL FROM:<notifications@clouddesk.example>"
Enter
Sleep 1s

Type "RCPT TO:<emily.carter@example.net>"
Enter
Sleep 1s

Type "DATA"
Enter
Sleep 1s

Type "From: CloudDesk <notifications@clouddesk.example>"
Enter
Type "To: Emily Carter <emily.carter@example.net>"
Enter
Type "Date: Mon, 17 Aug 2026 09:12:41 -0400"
Enter
Type "Subject: Welcome to CloudDesk"
Enter
Type "Message-ID: <welcome-7f3a21@clouddesk.example>"
Enter
Type "MIME-Version: 1.0"
Enter
Type "Content-Type: text/plain; charset='UTF-8'"
Enter
Enter

Type "Hi Emily,"
Enter
Enter

Type "Welcome to CloudDesk. Your account has been created successfully."
Enter
Enter
Type "You can now sign in and start setting up your workspace, invite team members, and configure your projects."
Enter
Enter
Type "Sign in: https://app.clouddesk.example/login"
Enter
Enter
Type "If you did not create this account, please contact our support team."
Enter
Enter
Type "Thanks,"
Enter
Type "The CloudDesk Team"

Enter
Type "."
Enter
Sleep 1s

Type "QUIT"
Enter
