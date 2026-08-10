#!/usr/bin/env python3
"""
search.py

Generates exactly 50 purpose-built test emails and sends them to a
maelsink SMTP listener, each one hand-crafted to exercise a *specific*
search pattern supported by the top search bar's `q` parameter.

Unlike scripts/assorted.py and scripts/html.py (which generate large,
randomized, "realistic traffic" datasets), this script is deterministic:
every email's From/To/Subject/Body/Tags is fixed and documented inline via
its `category` field, so the resulting inbox is a stable fixture for
manually or automatically exercising maelsink's search functionality.

Why this matters: maelsink's top search bar's `q` parameter is passed
verbatim into SQLite's FTS5 `MATCH` operator against a virtual table
indexing four columns (subject, from_addr, to_addrs, text_body) — see
internal/store/sqlite/migrations/0001_init.sql and
internal/store/sqlite/sqlite.go's ListFilter.Query handling. That means
the search bar transparently supports the *entire* FTS5 query grammar:
boolean AND/OR/NOT, phrase queries, column filters, prefix wildcards,
NEAR() proximity, and parenthesized grouping — not just naive substring
matching. This script's 50 emails are grouped into categories that each
demonstrate one such capability; internal-docs/SEARCH.md documents the
exact queries to run against them.

Only uses the Python standard library.

Usage:
    python3 scripts/search.py --host 127.0.0.1 --port 1025
    python3 scripts/search.py --dry-run   # print without sending

Requires a running maelsink SMTP listener (`maelsink serve`, default
127.0.0.1:1025) unless --dry-run is passed.
"""

import argparse
import io
import smtplib
import sys
import zipfile
from email.message import EmailMessage
from email.utils import formatdate, make_msgid


# ---------------------------------------------------------------------------
# Tiny stdlib-only fake-attachment builders (kept minimal — this script only
# needs "has an attachment", not realistic file contents).
# ---------------------------------------------------------------------------
def _fake_pdf_bytes(title: str) -> bytes:
    content_stream = f"BT /F1 18 Tf 72 700 Td ({title}) Tj ET".encode("latin-1", "replace")
    objects = [
        b"1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj",
        b"2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj",
        b"3 0 obj<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 5 0 R >> >> "
        b"/MediaBox [0 0 612 792] /Contents 4 0 R >>endobj",
        b"4 0 obj<< /Length %d >>stream\n" % len(content_stream) + content_stream + b"\nendstream endobj",
        b"5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj",
    ]
    buf = io.BytesIO()
    buf.write(b"%PDF-1.4\n")
    offsets = [0]
    for obj in objects:
        offsets.append(buf.tell())
        buf.write(obj + b"\n")
    xref_offset = buf.tell()
    buf.write(("xref\n0 %d\n" % (len(objects) + 1)).encode())
    buf.write(b"0000000000 65535 f \n")
    for off in offsets[1:]:
        buf.write(("%010d 00000 n \n" % off).encode())
    buf.write(b"trailer<< /Size %d /Root 1 0 R >>\n" % (len(objects) + 1))
    buf.write(b"startxref\n")
    buf.write(str(xref_offset).encode() + b"\n")
    buf.write(b"%%EOF")
    return buf.getvalue()


def _fake_zip_bytes(inner_name: str, inner_text: str) -> bytes:
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr(inner_name, inner_text)
    return buf.getvalue()


# ---------------------------------------------------------------------------
# The 50 emails. Each entry:
#   category:  which search pattern this email exists to demonstrate
#              (cross-referenced by internal-docs/SEARCH.md)
#   from_addr / to_addrs: envelope + header addresses
#   subject / text_body:  content indexed into messages_fts
#   tags:      X-Tag header values (repeatable) — sidebar tag filter, not
#              part of FTS matching, but combinable with `q` in a saved
#              search (see SEARCH.md's "Saved search" section)
#   attachment: optional (filename, bytes) — populates has_attachments,
#              combinable with `q` the same way tags are
# ---------------------------------------------------------------------------
EMAILS = [
    # -- Category: keyword — a single distinctive word anchors a simple,
    # unambiguous match. No boolean/phrase logic needed. --------------------
    dict(
        category="keyword",
        from_addr="billing@acmecorp.com",
        to_addrs=["ap@customer.example"],
        subject="Invoice #INV-2024-00987 is now overdue",
        text_body="This is a reminder that invoice INV-2024-00987 for $4,250.00 "
        "was due on 2026-07-01 and remains unpaid. Please remit payment at your "
        "earliest convenience to avoid a late fee.",
        tags=["invoice", "overdue"],
    ),
    dict(
        category="keyword",
        from_addr="hr@initech.com",
        to_addrs=["newhire@customer.example"],
        subject="Your onboarding checklist for week one",
        text_body="Welcome to the team! Please complete the attached onboarding "
        "checklist, including benefits enrollment and IT equipment pickup, by "
        "end of week one.",
        tags=["onboarding"],
    ),
    dict(
        category="keyword",
        from_addr="alerts@security.io",
        to_addrs=["oncall@customer.example"],
        subject="Server outage postmortem published",
        text_body="The postmortem for last night's outage of the payments "
        "cluster is now published. Root cause: a cascading connection pool "
        "exhaustion triggered by a slow downstream dependency.",
        tags=["incident"],
    ),
    dict(
        category="keyword",
        from_addr="noreply@identity.acmecorp.com",
        to_addrs=["alice@customer.example"],
        subject="Password reset requested",
        text_body="We received a request to reset the password for your "
        "account. If you did not make this request, you can safely ignore "
        "this email.",
        tags=["security"],
    ),
    dict(
        category="keyword",
        from_addr="board@initech.com",
        to_addrs=["exec-team@customer.example"],
        subject="Quarterly board meeting notes",
        text_body="Attached are the notes from this quarter's board meeting, "
        "covering revenue growth, headcount planning, and the upcoming "
        "product roadmap review.",
        tags=["meeting-notes"],
    ),
    # -- Category: phrase — pairs/triples of emails that share individual
    # words across unrelated messages, so an unquoted multi-word query
    # over-matches and a quoted phrase is required for precision. ----------
    dict(
        category="phrase",
        from_addr="dba@acmecorp.com",
        to_addrs=["platform-team@customer.example"],
        subject="Database migration completed successfully",
        text_body="The database migration to the new cluster completed "
        "successfully overnight with zero downtime. All read replicas have "
        "caught up and are serving traffic normally.",
        tags=["migration"],
    ),
    dict(
        category="phrase",
        from_addr="facilities@initech.com",
        to_addrs=["all-staff@customer.example"],
        subject="Office migration to the new building next month",
        text_body="As previously announced, our office migration to the new "
        "downtown building is scheduled for next month. Moving crews will "
        "begin packing shared equipment on the 3rd floor first.",
        tags=["facilities"],
    ),
    dict(
        category="phrase",
        from_addr="licensing@globex.io",
        to_addrs=["it-procurement@customer.example"],
        subject="Database license renewal due",
        text_body="Your enterprise database license is due for renewal. "
        "Please confirm the number of production and staging seats so we can "
        "generate the renewal quote.",
        tags=["licensing"],
    ),
    dict(
        category="phrase",
        from_addr="support@globex.io",
        to_addrs=["ops@customer.example"],
        subject="Scheduled maintenance window confirmed",
        text_body="This confirms the scheduled maintenance window for Saturday "
        "night. Expect brief connectivity blips as we roll nodes through the "
        "new configuration one at a time.",
        tags=["maintenance"],
    ),
    dict(
        category="phrase",
        from_addr="ops@globex.io",
        to_addrs=["customer-success@customer.example"],
        subject="Window replacement quote for the server room",
        text_body="Following the site visit, here is the quote for replacing "
        "the server room's exterior window with a properly sealed unit ahead "
        "of the scheduled maintenance work.",
        tags=["facilities"],
    ),
    dict(
        category="phrase",
        from_addr="pm@acmecorp.com",
        to_addrs=["engineering@customer.example"],
        subject="Q3 roadmap review scheduled",
        text_body="The Q3 roadmap review is scheduled for next Tuesday. Please "
        "come prepared to discuss the migration timeline and any blockers "
        "with the new database rollout.",
        tags=["roadmap"],
    ),
    # -- Category: boolean — vendor/urgency pairs for AND / OR / NOT (and the
    # `-term` shorthand for NOT) demonstrations. ----------------------------
    dict(
        category="boolean",
        from_addr="billing@acmecorp.com",
        to_addrs=["ap@customer.example"],
        subject="Invoice attached: Acme Corp consulting services",
        text_body="Please find attached this month's invoice for Acme Corp "
        "consulting services rendered in July. Standard net-30 terms apply.",
        tags=["invoice", "acme"],
        attachment=("acme-invoice-july.pdf", "pdf"),
    ),
    dict(
        category="boolean",
        from_addr="billing@globex.io",
        to_addrs=["ap@customer.example"],
        subject="Invoice attached: Globex support contract",
        text_body="Please find attached this month's invoice for the Globex "
        "annual support contract renewal, covering priority incident response.",
        tags=["invoice", "globex"],
        attachment=("globex-invoice-july.pdf", "pdf"),
    ),
    dict(
        category="boolean",
        from_addr="alerts@security.io",
        to_addrs=["oncall@customer.example"],
        subject="URGENT: certificate expiring in 24 hours",
        text_body="URGENT action required: the wildcard TLS certificate for "
        "the payments API expires in 24 hours. Rotate it before the deadline "
        "to avoid a hard outage.",
        tags=["security", "urgent"],
    ),
    dict(
        category="boolean",
        from_addr="alerts@security.io",
        to_addrs=["oncall@customer.example"],
        subject="Certificate renewed ahead of schedule",
        text_body="The wildcard TLS certificate for the payments API has been "
        "renewed ahead of schedule. No action is required from the on-call "
        "team at this time.",
        tags=["security"],
    ),
    dict(
        category="boolean",
        from_addr="sales@acmecorp.com",
        to_addrs=["procurement@customer.example"],
        subject="Following up on our proposal",
        text_body="Following up on the proposal we sent last week — happy to "
        "hop on a call if it would help answer any outstanding questions "
        "before your budget cycle closes.",
        tags=["sales"],
    ),
    dict(
        category="boolean",
        from_addr="sales@globex.io",
        to_addrs=["procurement@customer.example"],
        subject="Re: Following up on our proposal",
        text_body="Thanks for the quick turnaround. We're comparing your "
        "proposal against Acme's and should have a decision by Friday.",
        tags=["sales"],
    ),
    # -- Category: column — same keyword appears in different columns across
    # different emails, so subject:/from_addr:/to_addrs:/text_body: filters
    # return different, non-overlapping result sets. -----------------------
    dict(
        category="column",
        from_addr="noreply@shopify-clone.example",
        to_addrs=["customer@customer.example"],
        subject="Your Nimbus Cloud order has shipped",
        text_body="Good news — your recent order has shipped and is on its "
        "way. Tracking details are available in your account dashboard.",
        tags=["order-shipped"],
    ),
    dict(
        category="column",
        from_addr="nimbus-cloud@vendor.example",
        to_addrs=["it@customer.example"],
        subject="Scheduled maintenance for our platform",
        text_body="We're writing to let you know about upcoming scheduled "
        "maintenance. No action is required from your team.",
        tags=["maintenance"],
    ),
    dict(
        category="column",
        from_addr="notify@vendor.example",
        to_addrs=["nimbus-cloud-admins@customer.example"],
        subject="Weekly digest ready for review",
        text_body="Your weekly digest of account activity is ready. Log in to "
        "review usage trends and recommended optimizations.",
        tags=["digest"],
    ),
    dict(
        category="column",
        from_addr="reports@vendor.example",
        to_addrs=["finance@customer.example"],
        subject="Monthly usage report attached",
        text_body="Attached is the monthly usage report. Nimbus Cloud usage "
        "trended slightly down compared to last month across all regions.",
        tags=["report"],
        attachment=("usage-report.csv", "zip"),
    ),
    dict(
        category="column",
        from_addr="welcome@pixelforge.example",
        to_addrs=["daniel.reyes@customer.example"],
        subject="Welcome to PixelForge! Please confirm your email",
        text_body="Thanks for signing up. Please confirm your email address "
        "to activate your account and start your first project.",
        tags=["welcome", "signup-flow"],
    ),
    dict(
        category="column",
        from_addr="notifications@pixelforge.example",
        to_addrs=["team-updates@customer.example"],
        subject="Weekly project activity summary",
        text_body="Here is a summary of activity across your team's projects "
        "this week, including new comments, file uploads, and completed tasks.",
        tags=["digest"],
    ),
    # -- Category: prefix — shared word stems across emails, for `term*`
    # wildcard prefix matching. ---------------------------------------------
    dict(
        category="prefix",
        from_addr="billing@acmecorp.com",
        to_addrs=["finance@customer.example"],
        subject="Invoicing schedule change for next quarter",
        text_body="Starting next quarter, invoicing will move from monthly to "
        "bi-weekly cycles to smooth out cash flow on both sides.",
        tags=["invoice"],
    ),
    dict(
        category="prefix",
        from_addr="billing@globex.io",
        to_addrs=["finance@customer.example"],
        subject="Your account has been invoiced",
        text_body="Your account has been invoiced for the current billing "
        "period. A copy of the receipt is attached for your records.",
        tags=["invoice"],
        attachment=("receipt.pdf", "pdf"),
    ),
    dict(
        category="prefix",
        from_addr="devops@initech.com",
        to_addrs=["platform-team@customer.example"],
        subject="Configuration drift detected across staging nodes",
        text_body="Automated scanning detected configuration drift across "
        "three staging nodes. Recommend reapplying the baseline config before "
        "the next deploy.",
        tags=["ops"],
    ),
    dict(
        category="prefix",
        from_addr="devops@initech.com",
        to_addrs=["platform-team@customer.example"],
        subject="Reminder: configure your MFA before Friday",
        text_body="This is a reminder to configure multi-factor authentication "
        "on your account before Friday's security policy enforcement deadline.",
        tags=["security"],
    ),
    # -- Category: near — two terms appearing close together vs. far apart,
    # for NEAR(term1 term2, N) proximity matching. --------------------------
    dict(
        category="near",
        from_addr="support@globex.io",
        to_addrs=["customer@customer.example"],
        subject="Refund processed for your recent order",
        text_body="Your refund has been processed and should appear on your "
        "statement within 3-5 business days. Let us know if you have any "
        "questions about the refund.",
        tags=["refund"],
    ),
    dict(
        category="near",
        from_addr="support@globex.io",
        to_addrs=["customer@customer.example"],
        subject="Order update and unrelated account note",
        text_body="Your order has shipped and should arrive by Thursday. "
        "Separately, on an unrelated note, we noticed your billing address "
        "was recently updated; if that wasn't you, please contact support so "
        "we can review the refund policy just in case a mistaken charge needs "
        "attention.",
        tags=["order-shipped"],
    ),
    dict(
        category="near",
        from_addr="billing@acmecorp.com",
        to_addrs=["ap@customer.example"],
        subject="Payment failed for invoice INV-2024-01055",
        text_body="Your payment failed for invoice INV-2024-01055 due to an "
        "expired card on file. Please update your payment method to avoid "
        "service interruption.",
        tags=["invoice", "payment-failed"],
    ),
    dict(
        category="near",
        from_addr="billing@acmecorp.com",
        to_addrs=["ap@customer.example"],
        subject="Account summary and general notes",
        text_body="Here is your account summary for the quarter. On a general "
        "note unrelated to billing mechanics, we've also updated our support "
        "hours; separately, any payment issues should be reported so our team "
        "can help before the invoice becomes overdue.",
        tags=["account-summary"],
    ),
    # -- Category: grouping — parenthesized boolean expressions combining
    # OR and AND across the vendor/category emails defined above. ----------
    dict(
        category="grouping",
        from_addr="ar@acmecorp.com",
        to_addrs=["ap@customer.example"],
        subject="Receipt for your recent payment",
        text_body="Thank you for your payment. Attached is the receipt for "
        "your records. Please reach out if any line item needs clarification.",
        tags=["receipt", "acme"],
        attachment=("payment-receipt.pdf", "pdf"),
    ),
    dict(
        category="grouping",
        from_addr="ar@globex.io",
        to_addrs=["ap@customer.example"],
        subject="Receipt confirmation for annual contract",
        text_body="This confirms receipt of your annual contract payment. "
        "Your service will renew automatically unless canceled 30 days prior "
        "to the term end date.",
        tags=["receipt", "globex"],
    ),
    dict(
        category="grouping",
        from_addr="ar@initech.com",
        to_addrs=["ap@customer.example"],
        subject="Receipt for consulting hours — Initech",
        text_body="Attached is the receipt for consulting hours billed this "
        "cycle. Let us know if the hourly breakdown needs any adjustment.",
        tags=["receipt", "initech"],
        attachment=("initech-receipt.pdf", "pdf"),
    ),
    # -- Category: numeric — order/ticket/confirmation codes as searchable
    # tokens, including mixed alphanumeric IDs. -----------------------------
    dict(
        category="numeric",
        from_addr="orders@shopify-clone.example",
        to_addrs=["customer@customer.example"],
        subject="Order confirmation ORD-55219",
        text_body="Thanks for your order! Your order number is ORD-55219 and "
        "your estimated delivery date is August 16, 2026.",
        tags=["order-confirmation"],
    ),
    dict(
        category="numeric",
        from_addr="support@initech.com",
        to_addrs=["customer@customer.example"],
        subject="Support ticket TCK-10432 has been updated",
        text_body="Your support ticket TCK-10432 has been updated by our "
        "engineering team. A fix is scheduled for the next release.",
        tags=["support-ticket"],
    ),
    dict(
        category="numeric",
        from_addr="noreply@identity.acmecorp.com",
        to_addrs=["alice@customer.example"],
        subject="Your one-time verification code",
        text_body="Your one-time verification code is 482913. This code "
        "expires in 10 minutes. Do not share this code with anyone.",
        tags=["2fa"],
    ),
    dict(
        category="numeric",
        from_addr="billing@globex.io",
        to_addrs=["ap@customer.example"],
        subject="Invoice INV-2024-00764 payment confirmation",
        text_body="This confirms payment was received for invoice "
        "INV-2024-00764 in full. No further action is needed.",
        tags=["invoice", "payment-confirmed"],
    ),
    # -- Category: punctuation — hyphens, apostrophes, slashes: characters
    # that interact with FTS5's default unicode61 tokenizer and its unary
    # `-token` NOT-shorthand syntax. -----------------------------------------
    dict(
        category="punctuation",
        from_addr="hr@initech.com",
        to_addrs=["all-staff@customer.example"],
        subject="Follow-up: multi-factor authentication rollout",
        text_body="This is a follow-up on the multi-factor authentication "
        "rollout. If you haven't enrolled yet, please don't wait until the "
        "grace period ends.",
        tags=["security"],
    ),
    dict(
        category="punctuation",
        from_addr="it@initech.com",
        to_addrs=["all-staff@customer.example"],
        subject="Read/write access changes for the shared drive",
        text_body="We're updating read/write access permissions on the shared "
        "drive. Most staff will retain read-only access; write access now "
        "requires manager approval.",
        tags=["it-policy"],
    ),
    dict(
        category="punctuation",
        from_addr="support@globex.io",
        to_addrs=["customer@customer.example"],
        subject="We can't process your request yet",
        text_body="Unfortunately we can't process your request yet — we're "
        "missing a signed W-9 form. Once that's received we'll finish "
        "processing right away.",
        tags=["support"],
    ),
    dict(
        category="punctuation",
        from_addr="sales@acmecorp.com",
        to_addrs=["procurement@customer.example"],
        subject="Add-on pricing for the pro+ tier",
        text_body="Here's the add-on pricing for the pro+ tier, including the "
        "co-branded onboarding package and the always-on support add-on.",
        tags=["sales"],
    ),
    # -- Category: unicode — diacritics/non-ASCII content, to demonstrate
    # FTS5's default diacritic-insensitive matching. -----------------------
    dict(
        category="unicode",
        from_addr="events@meridianstudio.example",
        to_addrs=["maria.dossantos@customer.example"],
        subject="You're on the Meridian Studio waitlist!",
        text_body="Thanks for joining! Our café launch event in Zürich is "
        "planned for this résumé-only preview — naïve early access opens "
        "next week for waitlisted members.",
        tags=["waitlist-confirmation"],
    ),
    dict(
        category="unicode",
        from_addr="hr@initech.com",
        to_addrs=["candidate@customer.example"],
        subject="Interview follow-up: résumé received",
        text_body="Thank you for sending your résumé. Our recruiting team in "
        "Zürich will follow up within a week to schedule your interview.",
        tags=["recruiting"],
    ),
    dict(
        category="unicode",
        from_addr="marketing@pixelforge.example",
        to_addrs=["subscriber@customer.example"],
        subject="Café hours extended for the Zürich pop-up",
        text_body="Good news — café hours for our Zürich pop-up have been "
        "extended through the weekend. Naïve about the location? Check the "
        "map link below.",
        tags=["marketing"],
    ),
    # -- Category: tag-combo — content that pairs naturally with a sidebar
    # tag/attachment filter, motivating a *saved search* that bundles `q`
    # with `tag=`/`has_attachments=` rather than retyping both every time. --
    dict(
        category="tag-combo",
        from_addr="rina.patel@customer.example",
        to_addrs=["carlos.mendez@customer.example"],
        subject='Rina shared "Onboarding Guide.pdf" with you',
        text_body="Rina Patel shared a document with you: Onboarding "
        "Guide.pdf. Open the attachment to view the shared file.",
        tags=["document-shared"],
        attachment=("Onboarding_Guide.pdf", "pdf"),
    ),
    dict(
        category="tag-combo",
        from_addr="care@driftstorage.example",
        to_addrs=["grace.kimani@customer.example"],
        subject="Invitation: 1:1 Sync @ August 18, 2026 10:30 AM",
        text_body="You have been invited to a 1:1 Sync meeting. Add this "
        "event to your calendar using the attached invite.",
        tags=["calendar-invite"],
        attachment=("invite.ics", "zip"),
    ),
    dict(
        category="tag-combo",
        from_addr="orders@brightcart.example",
        to_addrs=["support@lumen-bank.example"],
        subject="Your Nimbus Cloud account has been temporarily locked",
        text_body="For your security, your account has been temporarily "
        "locked after multiple failed login attempts. Verify your identity "
        "with two-factor authentication to unlock it.",
        tags=["2fa", "account-locked"],
    ),
    dict(
        category="tag-combo",
        from_addr="receipts@cobaltmarket.example",
        to_addrs=["no-reply@auth.vertexsoft.example"],
        subject="Scheduled maintenance for Nimbus Cloud on August 13, 2026",
        text_body="We'll be performing scheduled maintenance on Nimbus Cloud. "
        "Expect brief service interruptions during the maintenance window.",
        tags=["maintenance-notice", "signup-flow", "smoke"],
    ),
    dict(
        category="tag-combo",
        from_addr="isabelle.roy@customer.example",
        to_addrs=["cart@pixelforge-shop.example"],
        subject="Your refund of $111.13 has been processed",
        text_body="Your refund of $111.13 has been processed and will appear "
        "on your original payment method within 5-7 business days.",
        tags=["refund-processed"],
    ),
]

assert len(EMAILS) == 50, f"expected exactly 50 emails, got {len(EMAILS)}"


def build_message(entry: dict) -> EmailMessage:
    msg = EmailMessage()
    msg["Subject"] = entry["subject"]
    msg["From"] = entry["from_addr"]
    msg["To"] = ", ".join(entry["to_addrs"])
    msg["Date"] = formatdate(localtime=True)
    msg["Message-ID"] = make_msgid()

    for tag in entry.get("tags", []):
        # Repeatable "X-Tag" header, one per tag — see
        # internal/smtp/mime.go's extractTags, which scans every "X-Tag"
        # line on a message (case-insensitive, preserving duplicates/order).
        msg.add_header("X-Tag", tag)

    msg.set_content(entry["text_body"])

    attachment = entry.get("attachment")
    if attachment:
        filename, kind = attachment
        if kind == "pdf":
            data = _fake_pdf_bytes(entry["subject"])
            msg.add_attachment(data, maintype="application", subtype="pdf", filename=filename)
        elif kind == "zip":
            data = _fake_zip_bytes("notes.txt", entry["text_body"])
            msg.add_attachment(data, maintype="application", subtype="zip", filename=filename)

    return msg


def main(argv=None):
    parser = argparse.ArgumentParser(
        description="Send 50 purpose-built search-pattern test emails to a maelsink SMTP listener."
    )
    parser.add_argument("--host", default="127.0.0.1", help="SMTP server host (default: 127.0.0.1)")
    parser.add_argument("--port", type=int, default=1025, help="SMTP server port (default: 1025)")
    parser.add_argument(
        "--dry-run", action="store_true", help="Print message summaries without connecting to an SMTP server."
    )
    parser.add_argument("--quiet", action="store_true", help="Suppress per-message logging.")
    args = parser.parse_args(argv)

    smtp = None
    if not args.dry_run:
        smtp = smtplib.SMTP(args.host, args.port)

    try:
        for i, entry in enumerate(EMAILS, start=1):
            msg = build_message(entry)
            if not args.quiet:
                tags = ",".join(entry.get("tags", [])) or "-"
                has_attachment = "yes" if entry.get("attachment") else "no"
                print(
                    f"[{i:2d}/50] category={entry['category']:<11} "
                    f"attachment={has_attachment:<3} tags={tags:<20} "
                    f"subject={entry['subject']!r}"
                )
            if not args.dry_run:
                smtp.send_message(msg)
    finally:
        if smtp is not None:
            smtp.quit()

    print(f"\nDone. {'Simulated' if args.dry_run else 'Sent'} {len(EMAILS)} email(s).")
    return 0


if __name__ == "__main__":
    sys.exit(main())
