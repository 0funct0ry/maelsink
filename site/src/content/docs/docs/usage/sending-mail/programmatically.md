---
title: Sending Mail Programmatically
description: Fully worked SMTP examples for popular languages and frameworks, all pointed at a local maelsink serve instance.
---

`maelsink serve` accepts mail on a plain SMTP listener (default `127.0.0.1:1025`, no auth, no TLS) from any client library that supports SMTP. An application's mail configuration can be pointed at that host:port during local development or CI; every message it sends is captured by maelsink instead of leaving the machine.

:::note
The Node/Nodemailer, Python/`smtplib`, and Go/`net/smtp` examples below were run against a real local `maelsink serve` instance while writing this page, and confirmed to arrive via `GET /api/v1/messages`. The remaining examples (Ruby, PHP, Java, .NET, Django) are written to match each library's documented API but weren't executed in this build's sandbox.
:::

## Node.js — Nodemailer

```js
const nodemailer = require("nodemailer");

async function main() {
  const transporter = nodemailer.createTransport({
    host: "127.0.0.1",
    port: 1025,
    secure: false,
    ignoreTLS: true,
  });

  const info = await transporter.sendMail({
    from: "app@example.com",
    to: "dev@example.com",
    subject: "Nodemailer live check",
    text: "Sent from the Nodemailer docs example",
  });

  console.log("sent:", info.messageId);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
```

Live-verified: running this against a local `maelsink serve` produced `sent: <...@example.com>`, and the message showed up in `GET /api/v1/messages` with subject `Nodemailer live check`.

## Python — `smtplib`

```python
import smtplib
from email.message import EmailMessage

msg = EmailMessage()
msg["From"] = "app@example.com"
msg["To"] = "dev@example.com"
msg["Subject"] = "smtplib live check"
msg.set_content("Sent from the Python smtplib docs example")

with smtplib.SMTP("127.0.0.1", 1025) as smtp:
    smtp.send_message(msg)

print("sent")
```

Live-verified the same way — the message arrived with subject `smtplib live check`.

## Go — `net/smtp`

```go
package main

import (
	"fmt"
	"net/smtp"
)

func main() {
	from := "app@example.com"
	to := []string{"dev@example.com"}
	msg := []byte("From: app@example.com\r\n" +
		"To: dev@example.com\r\n" +
		"Subject: net/smtp live check\r\n" +
		"\r\n" +
		"Sent from the Go net/smtp docs example\r\n")

	if err := smtp.SendMail("127.0.0.1:1025", nil, from, to, msg); err != nil {
		panic(err)
	}
	fmt.Println("sent")
}
```

`smtp.SendMail`'s third argument is the auth mechanism — pass `nil` for maelsink's default unauthenticated listener. Live-verified the same way — the message arrived with subject `net/smtp live check`.

## Ruby — `mail`

<!-- unverified: no ruby available in build sandbox -->
```ruby
require "mail"

Mail.defaults do
  delivery_method :smtp, address: "127.0.0.1", port: 1025, openssl_verify_mode: "none"
end

Mail.deliver do
  from    "app@example.com"
  to      "dev@example.com"
  subject "Ruby mail gem check"
  body    "Sent from the Ruby mail gem docs example"
end
```

## PHP — Symfony Mailer

<!-- unverified: no PHP available in build sandbox -->
```php
<?php

require "vendor/autoload.php";

use Symfony\Component\Mailer\Mailer;
use Symfony\Component\Mailer\Transport;
use Symfony\Component\Mime\Email;

$transport = Transport::fromDsn("smtp://127.0.0.1:1025");
$mailer = new Mailer($transport);

$email = (new Email())
    ->from("app@example.com")
    ->to("dev@example.com")
    ->subject("Symfony Mailer check")
    ->text("Sent from the Symfony Mailer docs example");

$mailer->send($email);
```

## Java — Jakarta Mail

<!-- unverified: no Java available in build sandbox -->
```java
import jakarta.mail.*;
import jakarta.mail.internet.*;
import java.util.Properties;

public class SendTest {
    public static void main(String[] args) throws MessagingException {
        Properties props = new Properties();
        props.put("mail.smtp.host", "127.0.0.1");
        props.put("mail.smtp.port", "1025");
        props.put("mail.smtp.auth", "false");

        Session session = Session.getInstance(props);
        Message message = new MimeMessage(session);
        message.setFrom(new InternetAddress("app@example.com"));
        message.setRecipients(Message.RecipientType.TO, InternetAddress.parse("dev@example.com"));
        message.setSubject("Jakarta Mail check");
        message.setText("Sent from the Jakarta Mail docs example");

        Transport.send(message);
    }
}
```

## .NET — MailKit

<!-- unverified: no .NET available in build sandbox -->
```csharp
using MailKit.Net.Smtp;
using MimeKit;

var message = new MimeMessage();
message.From.Add(MailboxAddress.Parse("app@example.com"));
message.To.Add(MailboxAddress.Parse("dev@example.com"));
message.Subject = "MailKit check";
message.Body = new TextPart("plain") { Text = "Sent from the MailKit docs example" };

using var client = new SmtpClient();
client.Connect("127.0.0.1", 1025, MailKit.Security.SecureSocketOptions.None);
client.Send(message);
client.Disconnect(true);
```

## Framework-native — Django

<!-- unverified: no Django available in build sandbox -->
Point Django's built-in SMTP email backend at maelsink in `settings.py`:

```python
EMAIL_BACKEND = "django.core.mail.backends.smtp.EmailBackend"
EMAIL_HOST = "127.0.0.1"
EMAIL_PORT = 1025
EMAIL_USE_TLS = False
EMAIL_USE_SSL = False
```

Then send normally with `send_mail`:

```python
from django.core.mail import send_mail

send_mail(
    "Django send_mail check",
    "Sent from the Django docs example",
    "app@example.com",
    ["dev@example.com"],
)
```

No code changes are needed elsewhere in the app — anything using Django's mail API transparently lands in maelsink for the duration of local dev/CI.
