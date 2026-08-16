// "Load example" content for the Composer (SPEC.md §7.7.4.1). Each entry
// provides both an EML and a JSON rendering of the same scenario so
// switching modes after loading an example still makes sense. Uses real
// registered template functions (fName, fEmail, uuid, etc. — see
// /compose-api/v1/functions) rather than invented names.
//
// internal/shell/builtin's `example` scenario data lives under
// internal/shell/builtin, which internal/compose must not import (SPEC.md
// §7.7.2's leaf-package rule permits internal/shell/tmpl only) — so this is
// a client-side equivalent, not a /compose-api/v1/example proxy. Revisit if
// a stateless proxy endpoint becomes worth adding later.

import type { AttachmentInput, TemplateFormat } from './composeApi'

export interface ExampleTemplate {
  id: string
  label: string
  eml: string
  json: string
  // emlAttachments is spliced in by the backend around the rendered eml
  // template (internal/compose/render.go's buildEMLWithAttachments) — the
  // json variant's attachments live inline in its own template text instead
  // (see jsonSpecHtmlWithAttachment below).
  emlAttachments?: AttachmentInput[]
}

function eml(headers: Record<string, string>, body: string): string {
  const headerLines = Object.entries(headers)
    .map(([k, v]) => `${k}: ${v}`)
    .join('\r\n')
  return `${headerLines}\r\n\r\n${body}`
}

function jsonSpec(from: string, subject: string, text: string): string {
  return JSON.stringify(
    { from, to: ['{{ .to }}'], subject, text },
    null,
    2,
  )
}

// emlHtml builds a raw single-part text/html message — the raw-EML editor
// has no structured way to build a MIME multipart/mixed envelope by hand,
// so (unlike jsonSpecHtmlWithAttachment below) there is no EML variant with
// a real attachment; a note in the body says so.
function emlHtml(headers: Record<string, string>, htmlBody: string): string {
  return eml({ ...headers, 'MIME-Version': '1.0', 'Content-Type': 'text/html; charset=utf-8' }, htmlBody)
}

function jsonSpecHtml(from: string, subject: string, html: string): string {
  return JSON.stringify({ from, to: ['{{ .to }}'], subject, html }, null, 2)
}

// jsonSpecHtmlWithAttachment sets attachments[0].path to a template
// expression that calls a file-generating function (e.g. {{ fPDF }}); the
// compose backend's /render and /send handlers template this field per
// request (internal/compose/render.go's renderAttachments) before
// spec.Build attaches whatever file it resolves to.
function jsonSpecHtmlWithAttachment(
  from: string,
  subject: string,
  html: string,
  attachmentPathExpr: string,
  filename: string,
): string {
  return JSON.stringify(
    {
      from,
      to: ['{{ .to }}'],
      subject,
      html,
      attachments: [{ path: attachmentPathExpr, filename }],
    },
    null,
    2,
  )
}

export const exampleTemplates: ExampleTemplate[] = [
  {
    id: 'welcome',
    label: 'Welcome email',
    eml: eml(
      { From: '{{ fName }} <{{ fEmail }}>', To: '{{ .to }}', Subject: 'Welcome, {{ fFirstName }}!' },
      'Hi {{ fFirstName }},\n\nYour account {{ uuid }} was just created.\n\nThanks,\nThe Team',
    ),
    json: jsonSpec(
      '{{ fEmail }}',
      'Welcome, {{ fFirstName }}!',
      'Hi {{ fFirstName }}, your account {{ uuid }} was just created.',
    ),
  },
  {
    id: 'password-reset',
    label: 'Password reset',
    eml: eml(
      { From: 'Security <security@{{ fDomain }}>', To: '{{ .to }}', Subject: 'Reset your password' },
      'Hi {{ fFirstName }},\n\nWe received a request to reset your password. Use the code below within 15 minutes:\n\n  {{ randInt 100000 999999 }}\n\nIf you did not request this, ignore this email.\n\nThanks,\nSecurity Team',
    ),
    json: jsonSpec(
      'security@{{ fDomain }}',
      'Reset your password',
      'Hi {{ fFirstName }}, your reset code is {{ randInt 100000 999999 }}. It expires in 15 minutes.',
    ),
  },
  {
    id: 'email-verification',
    label: 'Email verification (OTP)',
    eml: eml(
      { From: 'noreply@{{ fDomain }}', To: '{{ .to }}', Subject: 'Verify your email address' },
      'Hi {{ fFirstName }},\n\nEnter this code to verify your email:\n\n  {{ randInt 100000 999999 }}\n\nThis code expires in 10 minutes.',
    ),
    json: jsonSpec(
      'noreply@{{ fDomain }}',
      'Verify your email address',
      'Hi {{ fFirstName }}, your verification code is {{ randInt 100000 999999 }} (expires in 10 minutes).',
    ),
  },
  {
    id: 'order-confirmation',
    label: 'Order confirmation',
    eml: eml(
      { From: 'orders@{{ fDomain }}', To: '{{ .to }}', Subject: 'Order confirmed — #{{ randInt 10000 99999 }}' },
      'Hi {{ fFirstName }},\n\nThanks for your order! Here are the details:\n\n  Order #: {{ randInt 10000 99999 }}\n  Item: {{ oneOf "Wireless Keyboard" "Standing Desk" "Noise-Cancelling Headphones" "Mechanical Mouse" }}\n  Total: ${{ randFloat 10 500 2 }}\n\nWe will email you again once it ships.',
    ),
    json: jsonSpec(
      'orders@{{ fDomain }}',
      'Order confirmed — #{{ randInt 10000 99999 }}',
      'Hi {{ fFirstName }}, thanks for your order! Total: ${{ randFloat 10 500 2 }}.',
    ),
  },
  {
    id: 'shipping-notification',
    label: 'Shipping notification',
    eml: eml(
      { From: 'shipping@{{ fDomain }}', To: '{{ .to }}', Subject: 'Your order has shipped!' },
      'Hi {{ fFirstName }},\n\nGood news — your order is on its way.\n\n  Carrier: {{ oneOf "UPS" "FedEx" "USPS" "DHL" }}\n  Tracking #: {{ ulid }}\n  Estimated delivery: {{ date "Jan 2, 2006" now }}\n\nTrack your package any time from your account.',
    ),
    json: jsonSpec(
      'shipping@{{ fDomain }}',
      'Your order has shipped!',
      'Hi {{ fFirstName }}, your order shipped via {{ oneOf "UPS" "FedEx" "USPS" "DHL" }}, tracking # {{ ulid }}.',
    ),
  },
  {
    id: 'payment-receipt',
    label: 'Payment receipt',
    eml: eml(
      { From: 'billing@{{ fDomain }}', To: '{{ .to }}', Subject: 'Receipt for your recent payment' },
      'Hi {{ fFirstName }},\n\nThis confirms your payment was processed successfully.\n\n  Amount: ${{ randFloat 5 200 2 }}\n  Card: **** {{ randInt 1000 9999 }}\n  Date: {{ rfc2822Date }}\n\nA copy of this receipt has been saved to your account.',
    ),
    json: jsonSpec(
      'billing@{{ fDomain }}',
      'Receipt for your recent payment',
      'Hi {{ fFirstName }}, we processed your payment of ${{ randFloat 5 200 2 }} on {{ rfc2822Date }}.',
    ),
  },
  {
    id: 'account-alert',
    label: 'Security alert (new sign-in)',
    eml: eml(
      { From: 'alerts@{{ fDomain }}', To: '{{ .to }}', Subject: 'New sign-in to your account' },
      'Hi {{ fFirstName }},\n\nWe noticed a new sign-in to your account:\n\n  Device: {{ fUserAgent }}\n  Location: {{ fCity }}, {{ fState }}\n  IP address: {{ fIPv4 }}\n  Time: {{ rfc2822Date }}\n\nIf this was not you, please reset your password immediately.',
    ),
    json: jsonSpec(
      'alerts@{{ fDomain }}',
      'New sign-in to your account',
      'Hi {{ fFirstName }}, we noticed a new sign-in from {{ fCity }}, {{ fState }} ({{ fIPv4 }}) at {{ rfc2822Date }}.',
    ),
  },
  {
    id: 'newsletter',
    label: 'Newsletter / product update',
    eml: eml(
      { From: '{{ fCompany }} <news@{{ fDomain }}>', To: '{{ .to }}', Subject: "This month's product update" },
      "Hi {{ fFirstName }},\n\nHere's what's new this month:\n\n  - {{ fSentence }}\n  - {{ fSentence }}\n  - {{ fSentence }}\n\nRead the full changelog on our blog.\n\n— The {{ fCompany }} Team",
    ),
    json: jsonSpec(
      'news@{{ fDomain }}',
      "This month's product update",
      'Hi {{ fFirstName }}, here is what is new this month: {{ fSentence }}',
    ),
  },
  {
    id: 'meeting-reminder',
    label: 'Meeting reminder',
    eml: eml(
      { From: 'calendar@{{ fDomain }}', To: '{{ .to }}', Subject: 'Reminder: {{ fJobTitle }} sync tomorrow' },
      'Hi {{ fFirstName }},\n\nThis is a reminder for your upcoming meeting:\n\n  What: {{ fJobTitle }} sync\n  When: {{ date "Mon, Jan 2 at 3:04 PM" now }}\n  Where: {{ oneOf "Conference Room A" "Zoom" "Google Meet" }}\n\nSee you there!',
    ),
    json: jsonSpec(
      'calendar@{{ fDomain }}',
      'Reminder: {{ fJobTitle }} sync tomorrow',
      'Hi {{ fFirstName }}, reminder for your meeting on {{ date "Mon, Jan 2 at 3:04 PM" now }}.',
    ),
  },
  {
    id: 'support-ticket',
    label: 'Support ticket update',
    eml: eml(
      { From: 'support@{{ fDomain }}', To: '{{ .to }}', Subject: 'Update on ticket #{{ randInt 1000 9999 }}' },
      'Hi {{ fFirstName }},\n\nYour support ticket has been updated:\n\n  Ticket #: {{ randInt 1000 9999 }}\n  Status: {{ oneOf "Open" "In Progress" "Waiting on you" "Resolved" }}\n  Assigned to: {{ fName }}\n\nReply to this email to add a comment to your ticket.',
    ),
    json: jsonSpec(
      'support@{{ fDomain }}',
      'Update on ticket #{{ randInt 1000 9999 }}',
      'Hi {{ fFirstName }}, your ticket status is now {{ oneOf "Open" "In Progress" "Waiting on you" "Resolved" }}.',
    ),
  },

  // --- HTML-bodied examples (no attachments) ---
  {
    id: 'html-newsletter',
    label: 'HTML: Newsletter',
    eml: emlHtml(
      { From: '{{ fCompany }} <news@{{ fDomain }}>', To: '{{ .to }}', Subject: "This month's highlights" },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">' +
        '<h1 style="color: #635bff;">{{ fCompany }}</h1>' +
        '<p>Hi {{ fFirstName }},</p>' +
        '<p>Here is what shipped this month:</p>' +
        '<ul><li>{{ fSentence }}</li><li>{{ fSentence }}</li><li>{{ fSentence }}</li></ul>' +
        '<p><a href="https://{{ fDomain }}/changelog" style="color: #635bff;">Read the full changelog &rarr;</a></p>' +
        '<p style="color: #6b7280; font-size: 12px;">— The {{ fCompany }} Team</p>' +
        '</div>',
    ),
    json: jsonSpecHtml(
      'news@{{ fDomain }}',
      "This month's highlights",
      '<h1>{{ fCompany }}</h1><p>Hi {{ fFirstName }}, here is what shipped this month: {{ fSentence }}</p>',
    ),
  },
  {
    id: 'html-invoice',
    label: 'HTML: Invoice',
    eml: emlHtml(
      { From: 'billing@{{ fDomain }}', To: '{{ .to }}', Subject: 'Invoice #{{ randInt 10000 99999 }}' },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">' +
        '<h2>Invoice #{{ randInt 10000 99999 }}</h2>' +
        '<p>Billed to: {{ fFirstName }} {{ fLastName }}<br/>{{ fAddress }}</p>' +
        '<table style="width: 100%; border-collapse: collapse;">' +
        '<tr style="border-bottom: 1px solid #e3e5ea;"><td>{{ fJobTitle }} plan</td><td style="text-align: right;">${{ randFloat 20 200 2 }}</td></tr>' +
        '<tr><td><strong>Total</strong></td><td style="text-align: right;"><strong>${{ randFloat 20 200 2 }}</strong></td></tr>' +
        '</table>' +
        '<p style="color: #6b7280; font-size: 12px;">Issued {{ rfc2822Date }}</p>' +
        '</div>',
    ),
    json: jsonSpecHtml(
      'billing@{{ fDomain }}',
      'Invoice #{{ randInt 10000 99999 }}',
      '<h2>Invoice #{{ randInt 10000 99999 }}</h2><p>Billed to {{ fFirstName }} {{ fLastName }} — Total: ${{ randFloat 20 200 2 }}</p>',
    ),
  },
  {
    id: 'html-product-launch',
    label: 'HTML: Product launch',
    eml: emlHtml(
      { From: '{{ fCompany }} <hello@{{ fDomain }}>', To: '{{ .to }}', Subject: 'Introducing our newest feature' },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto; text-align: center;">' +
        '<h1>Something new just launched</h1>' +
        '<p>Hi {{ fFirstName }}, {{ fSentence }}</p>' +
        '<p><a href="https://{{ fDomain }}" style="background: #635bff; color: white; padding: 10px 20px; border-radius: 6px; text-decoration: none;">Try it now</a></p>' +
        '</div>',
    ),
    json: jsonSpecHtml(
      'hello@{{ fDomain }}',
      'Introducing our newest feature',
      '<h1>Something new just launched</h1><p>Hi {{ fFirstName }}, {{ fSentence }}</p>',
    ),
  },
  {
    id: 'html-event-invite',
    label: 'HTML: Event invitation',
    eml: emlHtml(
      { From: 'events@{{ fDomain }}', To: '{{ .to }}', Subject: "You're invited: {{ fCompany }} Summit" },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto; border: 1px solid #e3e5ea; border-radius: 8px; padding: 24px;">' +
        '<h2>{{ fCompany }} Summit</h2>' +
        '<p>Hi {{ fFirstName }}, you are invited to join us:</p>' +
        '<p><strong>When:</strong> {{ date "Monday, Jan 2 at 3:04 PM" now }}<br/>' +
        '<strong>Where:</strong> {{ fCity }}, {{ fState }}</p>' +
        '<p><a href="https://{{ fDomain }}/rsvp" style="color: #635bff;">RSVP now &rarr;</a></p>' +
        '</div>',
    ),
    json: jsonSpecHtml(
      'events@{{ fDomain }}',
      "You're invited: {{ fCompany }} Summit",
      '<h2>{{ fCompany }} Summit</h2><p>Hi {{ fFirstName }}, join us {{ date "Monday, Jan 2 at 3:04 PM" now }} in {{ fCity }}.</p>',
    ),
  },
  {
    id: 'html-password-reset',
    label: 'HTML: Password reset (styled)',
    eml: emlHtml(
      { From: 'Security <security@{{ fDomain }}>', To: '{{ .to }}', Subject: 'Reset your password' },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">' +
        '<h2>Reset your password</h2>' +
        '<p>Hi {{ fFirstName }}, click below to choose a new password. This link expires in 15 minutes.</p>' +
        '<p><a href="https://{{ fDomain }}/reset/{{ uuid }}" style="background: #df1b41; color: white; padding: 10px 20px; border-radius: 6px; text-decoration: none;">Reset password</a></p>' +
        '<p style="color: #6b7280; font-size: 12px;">If you did not request this, you can safely ignore this email.</p>' +
        '</div>',
    ),
    json: jsonSpecHtml(
      'security@{{ fDomain }}',
      'Reset your password',
      '<h2>Reset your password</h2><p>Hi {{ fFirstName }}, use this link: https://{{ fDomain }}/reset/{{ uuid }}</p>',
    ),
  },

  // --- HTML-bodied examples with a generated attachment ---
  // Both the eml and json variants attach a real generated file: eml via
  // the top-level emlAttachments field (spliced in server-side by
  // buildEMLWithAttachments), json via its own inline attachments field
  // (jsonSpecHtmlWithAttachment).
  {
    id: 'html-report-pdf',
    label: 'HTML: Monthly report (+ PDF attachment)',
    eml: emlHtml(
      { From: 'reports@{{ fDomain }}', To: '{{ .to }}', Subject: 'Your monthly report is ready' },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">' +
        '<h2>Your monthly report</h2>' +
        '<p>Hi {{ fFirstName }}, your report for {{ date "January 2006" now }} is attached.</p>' +
        '</div>',
    ),
    emlAttachments: [{ path: '{{ fPDF }}', filename: 'monthly-report.pdf' }],
    json: jsonSpecHtmlWithAttachment(
      'reports@{{ fDomain }}',
      'Your monthly report is ready',
      '<h2>Your monthly report</h2><p>Hi {{ fFirstName }}, your report for {{ date "January 2006" now }} is attached.</p>',
      '{{ fPDF }}',
      'monthly-report.pdf',
    ),
  },
  {
    id: 'html-invoice-csv',
    label: 'HTML: Invoice (+ CSV attachment)',
    eml: emlHtml(
      { From: 'billing@{{ fDomain }}', To: '{{ .to }}', Subject: 'Invoice #{{ randInt 10000 99999 }} — line items attached' },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">' +
        '<h2>Invoice #{{ randInt 10000 99999 }}</h2>' +
        '<p>Hi {{ fFirstName }}, your itemized invoice (CSV) is attached. Total: ${{ randFloat 20 500 2 }}.</p>' +
        '</div>',
    ),
    emlAttachments: [{ path: '{{ fCSV }}', filename: 'invoice-items.csv' }],
    json: jsonSpecHtmlWithAttachment(
      'billing@{{ fDomain }}',
      'Invoice #{{ randInt 10000 99999 }} — line items attached',
      '<h2>Invoice #{{ randInt 10000 99999 }}</h2><p>Hi {{ fFirstName }}, your itemized invoice (CSV) is attached. Total: ${{ randFloat 20 500 2 }}.</p>',
      '{{ fCSV }}',
      'invoice-items.csv',
    ),
  },
  {
    id: 'html-photo-share',
    label: 'HTML: Photo share (+ PNG attachment)',
    eml: emlHtml(
      { From: '{{ fFirstName }} <{{ fEmail }}>', To: '{{ .to }}', Subject: '{{ fFirstName }} shared a photo with you' },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">' +
        '<h2>{{ fFirstName }} shared a photo with you</h2>' +
        '<p>{{ fSentence }}</p>' +
        '</div>',
    ),
    emlAttachments: [{ path: '{{ fPNG }}', filename: 'photo.png' }],
    json: jsonSpecHtmlWithAttachment(
      '{{ fEmail }}',
      '{{ fFirstName }} shared a photo with you',
      '<h2>{{ fFirstName }} shared a photo with you</h2><p>{{ fSentence }}</p>',
      '{{ fPNG }}',
      'photo.png',
    ),
  },
  {
    id: 'html-data-export-xlsx',
    label: 'HTML: Data export (+ XLSX attachment)',
    eml: emlHtml(
      { From: 'exports@{{ fDomain }}', To: '{{ .to }}', Subject: 'Your data export is ready' },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">' +
        '<h2>Your data export is ready</h2>' +
        '<p>Hi {{ fFirstName }}, the export you requested on {{ rfc2822Date }} is attached.</p>' +
        '</div>',
    ),
    emlAttachments: [{ path: '{{ fXLSX }}', filename: 'export.xlsx' }],
    json: jsonSpecHtmlWithAttachment(
      'exports@{{ fDomain }}',
      'Your data export is ready',
      '<h2>Your data export is ready</h2><p>Hi {{ fFirstName }}, the export you requested on {{ rfc2822Date }} is attached.</p>',
      '{{ fXLSX }}',
      'export.xlsx',
    ),
  },
  {
    id: 'html-backup-zip',
    label: 'HTML: Backup notification (+ ZIP attachment)',
    eml: emlHtml(
      { From: 'backups@{{ fDomain }}', To: '{{ .to }}', Subject: 'Your backup completed successfully' },
      '<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">' +
        '<h2>Backup completed</h2>' +
        '<p>Hi {{ fFirstName }}, your backup from {{ rfc2822Date }} is attached.</p>' +
        '</div>',
    ),
    emlAttachments: [{ path: '{{ fZIP }}', filename: 'backup.zip' }],
    json: jsonSpecHtmlWithAttachment(
      'backups@{{ fDomain }}',
      'Your backup completed successfully',
      '<h2>Backup completed</h2><p>Hi {{ fFirstName }}, your backup from {{ rfc2822Date }} is attached.</p>',
      '{{ fZIP }}',
      'backup.zip',
    ),
  },
]

export function exampleContent(id: string, format: TemplateFormat): string | undefined {
  const example = exampleTemplates.find((e) => e.id === id)
  if (!example) return undefined
  return format === 'eml' ? example.eml : example.json
}

// exampleEmlAttachments returns the attachments to load alongside an
// example's eml content (empty for examples with none, or when loading the
// json variant, whose attachments live inline in its own template text).
export function exampleEmlAttachments(id: string): AttachmentInput[] {
  return exampleTemplates.find((e) => e.id === id)?.emlAttachments ?? []
}
