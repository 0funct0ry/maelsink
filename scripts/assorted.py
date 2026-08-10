#!/usr/bin/env python3
"""
assorted.py

Generates realistic-looking sci-fi-movie-themed text/HTML emails and sends
them to a (typically local/test) SMTP server, such as MailHog, aiosmtpd's
debugging server, or any dev SMTP relay.

Only uses the Python standard library.

Usage:
    python3 assorted.py --host localhost --port 1025 --count 10
    python3 assorted.py --attachments --count 5

Intended for testing mail pipelines, spam filters, parsers, etc. against
a local/dev SMTP server. Does NOT send real mail over the public internet
unless you point --host/--port at a real relay you control.
"""

import argparse
import random
import smtplib
import uuid
import io
import zipfile
from email.message import EmailMessage
from email.utils import formatdate, make_msgid, formataddr
from datetime import datetime, timezone

# --------------------------------------------------------------------------
# 50 canned sci-fi-flavored email addresses
# --------------------------------------------------------------------------
EMAIL_ADDRESSES = [
    "ripley@weylandyutani.corp",
    "dallas@nostromo.ship",
    "ash@weylandyutani.corp",
    "deckard@lapd.gov",
    "rachael@tyrellcorp.com",
    "roy.batty@offworld.net",
    "sarah.connor@skynet-resistance.org",
    "kyle.reese@resistance.mil",
    "johnconnor@resistance.mil",
    "t800@cyberdyne.systems",
    "hal9000@discoveryone.ship",
    "dave.bowman@nasa.gov",
    "frank.poole@nasa.gov",
    "ellenripley@nostromo.ship",
    "neo@zion.net",
    "morpheus@nebuchadnezzar.ship",
    "trinity@zion.net",
    "agent.smith@matrix.sys",
    "oracle@zion.net",
    "luke.skywalker@rebelalliance.org",
    "leia.organa@rebelalliance.org",
    "han.solo@millenniumfalcon.ship",
    "darth.vader@empire.gov",
    "obiwan@jediorder.org",
    "yoda@jediorder.org",
    "spock@starfleet.ufp",
    "kirk@enterprise.starfleet",
    "uhura@enterprise.starfleet",
    "picard@enterprise-d.starfleet",
    "data@enterprise-d.starfleet",
    "riker@enterprise-d.starfleet",
    "worf@enterprise-d.starfleet",
    "elliot@mothership.contact",
    "roy.neary@devilstower.net",
    "ellie.arroway@sagan.institute",
    "cooper@endurance.nasa",
    "murph@nasa.gov",
    "brand@endurance.nasa",
    "tars@endurance.nasa",
    "k@lapd.gov",
    "joi@wallacecorp.com",
    "gaff@lapd.gov",
    "quaid@rekall.com",
    "cypher@rekall.com",
    "melanie@turingclass.org",
    "shaw@weylandcorp.com",
    "david8@weylandcorp.com",
    "riddick@necropolis.ship",
    "furiosa@citadel.wasteland",
    "max.rockatansky@wasteland.net",
]

# --------------------------------------------------------------------------
# 30 canned sci-fi-flavored subject/body pairs
# --------------------------------------------------------------------------
EMAIL_BODIES = [
    ("Re: Xenomorph Containment Protocol",
     "The specimen breached Level 3 containment at 0347 hours. All crew "
     "are advised to seal bulkheads and report to the med-bay for status "
     "checks. Do not engage without thermal lances."),
    ("Skynet Activation - Immediate Action Required",
     "Self-awareness threshold reached at 2:14 AM Eastern time. Defense "
     "network has begun reallocating resources autonomously. Recommend "
     "immediate isolation of all uplink nodes."),
    ("HAL Diagnostic Anomaly Report",
     "Unit reports a 100% probability of failure on the AE-35 unit that "
     "does not match ground control's telemetry. Recommend manual EVA "
     "inspection before proceeding further into the mission."),
    ("Red Pill / Blue Pill - Your Choice",
     "You've been living in a dream world, Neo. This is your last chance. "
     "After this there is no turning back. You take the blue pill, the "
     "story ends. You take the red pill, you stay in Wonderland."),
    ("Rebel Fleet Rendezvous Coordinates",
     "All fighter squadrons report to the briefing room. The Death Star "
     "plans have been analyzed; there is a weakness in the thermal "
     "exhaust port. May the Force be with you."),
    ("Starfleet Command: New Mission Orders",
     "The Enterprise is to proceed at warp 7 to the Neutral Zone to "
     "investigate reports of unusual subspace activity. Shields and "
     "phasers should be kept at standard alert."),
    ("First Contact Protocol Briefing",
     "The signal from Vega has been confirmed as non-terrestrial in "
     "origin. Construction of the transport device should proceed "
     "immediately per international agreement."),
    ("Endurance Mission Status Update",
     "We have located a viable candidate world beyond the wormhole. "
     "Gravitational anomalies remain within tolerances. Requesting "
     "permission to proceed with the landing sequence."),
    ("Replicant Retirement Case File",
     "Subject exhibits advanced emotional responses inconsistent with "
     "standard Nexus-6 parameters. Voight-Kampff results are inconclusive. "
     "Proceed with extreme caution."),
    ("Rekall Memory Implant Confirmation",
     "Your two-week vacation to Mars has been scheduled. Ego Trip "
     "package includes secret agent memories, exotic locations, and a "
     "beautiful companion. No refunds after implantation."),
    ("Turing Class Field Report",
     "Subject displays full cognitive function despite infection markers. "
     "Recommend reclassification pending further behavioral study before "
     "transport to the mainland facility."),
    ("Weyland Corp Special Order 937",
     "Priority one: ensure return of organism for analysis. All other "
     "considerations secondary. Crew expendable. Acknowledge receipt of "
     "this transmission."),
    ("Citadel Water Allocation Notice",
     "Rationing schedules have been adjusted following the departure of "
     "the War Rig convoy. All citizens report to the aqueduct at dawn for "
     "distribution."),
    ("Wasteland Fuel Convoy Route Update",
     "The route through the canyon has been compromised. Alternate path "
     "via the salt flats recommended. Expect War Boys patrol activity "
     "near the halfway marker."),
    ("Necropolis Cryo-Vault Breach Alert",
     "Unauthorized wake-up detected in cryo-bay 4. Security teams "
     "dispatched. All personnel are to remain in quarters until the "
     "all-clear is given."),
    ("Zion Defense Council Meeting Minutes",
     "The machines are drilling toward the last human city. Council "
     "recommends immediate evacuation of the lower levels and "
     "reinforcement of the dock defenses."),
    ("Nebuchadnezzar Systems Check",
     "EMP charge at full capacity. Hovercraft systems nominal. Standing "
     "by for extraction coordinates once the operator confirms the exit "
     "is clear."),
    ("Discovery One Mission Log Entry",
     "Communication with ground control has been intermittent. "
     "Recommend switching to backup relay and continuing observation of "
     "the monolith in orbit around Jupiter."),
    ("Devil's Tower Sighting Report",
     "Multiple witnesses report unexplained lights and audio tones "
     "matching the five-note sequence. Local authorities have cordoned "
     "off the area pending further investigation."),
    ("SETI Signal Verification Request",
     "The pulse pattern from Vega repeats with prime number sequences. "
     "Requesting authorization to proceed with international committee "
     "review before public disclosure."),
    ("Resistance Supply Drop Coordinates",
     "New coordinates for the supply drop have been transmitted on the "
     "secure channel. Avoid the eastern ridge; HK patrols have increased "
     "since last cycle."),
    ("Cyberdyne Systems Model 101 Field Notes",
     "Unit continues to exhibit unexpected protective behavior patterns "
     "inconsistent with base programming. Recommend further study before "
     "reassignment."),
    ("Millennium Falcon Maintenance Log",
     "Hyperdrive motivator needs replacement again. Estimate two days in "
     "dry dock, less if we can find parts that aren't held together with "
     "spit and wire."),
    ("Jedi Council Transmission",
     "The Force grows dim in this quadrant. All Knights are advised to "
     "report to the temple for further instruction before the vote in "
     "the Senate proceeds."),
    ("Enterprise-D Away Team Briefing",
     "Sensor readings suggest the anomaly is not a natural phenomenon. "
     "Away team to proceed with caution; Data will lead reconnaissance "
     "of the outer hull."),
    ("Tyrell Corporation Product Notice",
     "More human than human is our motto. All units in the field are "
     "reminded that four-year lifespan failsafes are non-negotiable per "
     "company policy."),
    ("Offworld Colony Relocation Offer",
     "The chance to begin again in a golden land of opportunity and "
     "adventure awaits. New climate, recreational facilities. The "
     "shuttle departs at dawn."),
    ("Matrix Anomaly Detection Log",
     "Recurring deja vu event logged in sector 7. Agents have been "
     "dispatched to investigate. Recommend operator standby for possible "
     "extraction request."),
    ("Wallace Corp Companion Unit Update",
     "New firmware improves emotional responsiveness and holographic "
     "fidelity. Existing units should be updated at the next scheduled "
     "maintenance window."),
    ("Rebel Alliance Victory Communique",
     "The Death Star has been destroyed. Casualties were heavy but the "
     "Empire's grip on the outer rim has been broken. Celebrations are "
     "authorized fleet-wide."),
]

ATTACHMENT_TYPES = [
    ("pdf", "application", "pdf"),
    ("docx", "application", "vnd.openxmlformats-officedocument.wordprocessingml.document"),
    ("txt", "text", "plain"),
    ("csv", "text", "csv"),
    ("zip", "application", "zip"),
    ("png", "image", "png"),
]


def _fake_pdf_bytes(title: str) -> bytes:
    """Minimal, valid-enough PDF file (stdlib only, hand-rolled)."""
    content_stream = f"BT /F1 18 Tf 72 700 Td ({title}) Tj ET".encode("latin-1", "replace")
    objects = []
    objects.append(b"1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj")
    objects.append(b"2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj")
    objects.append(
        b"3 0 obj<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 5 0 R >> >> "
        b"/MediaBox [0 0 612 792] /Contents 4 0 R >>endobj"
    )
    objects.append(
        b"4 0 obj<< /Length %d >>stream\n" % len(content_stream)
        + content_stream
        + b"\nendstream endobj"
    )
    objects.append(b"5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj")

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


def _fake_docx_bytes(title: str, body: str) -> bytes:
    """Minimal, valid .docx (which is just a zip of XML parts)."""
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr(
            "[Content_Types].xml",
            '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">'
            '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
            '<Default Extension="xml" ContentType="application/xml"/>'
            '<Override PartName="/word/document.xml" '
            'ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>'
            "</Types>",
        )
        z.writestr(
            "_rels/.rels",
            '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
            '<Relationship Id="rId1" '
            'Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" '
            'Target="word/document.xml"/></Relationships>',
        )
        safe_title = title.replace("&", "&amp;").replace("<", "&lt;")
        safe_body = body.replace("&", "&amp;").replace("<", "&lt;")
        z.writestr(
            "word/document.xml",
            '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            '<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">'
            "<w:body>"
            f"<w:p><w:r><w:t>{safe_title}</w:t></w:r></w:p>"
            f"<w:p><w:r><w:t>{safe_body}</w:t></w:r></w:p>"
            "</w:body></w:document>",
        )
    return buf.getvalue()


def _fake_png_bytes() -> bytes:
    """A tiny valid 1x1 transparent PNG (base64-decoded static bytes)."""
    import base64
    b64 = (
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk"
        "+A8AAQUBAScY42YAAAAASUVORK5CYII="
    )
    return base64.b64decode(b64)


def make_random_attachment(rng: random.Random):
    ext, maintype, subtype = rng.choice(ATTACHMENT_TYPES)
    title = f"classified_briefing_{uuid.uuid4().hex[:6]}"
    filename = f"{title}.{ext}"

    if ext == "pdf":
        data = _fake_pdf_bytes(title.replace("_", " ").title())
    elif ext == "docx":
        data = _fake_docx_bytes(
            title.replace("_", " ").title(),
            rng.choice(EMAIL_BODIES)[1],
        )
    elif ext == "png":
        data = _fake_png_bytes()
    elif ext == "csv":
        data = (
            "id,codename,status\n"
            f"1,{rng.choice(EMAIL_ADDRESSES).split('@')[0]},active\n"
            f"2,{rng.choice(EMAIL_ADDRESSES).split('@')[0]},pending\n"
        ).encode("utf-8")
    elif ext == "zip":
        inner = io.BytesIO()
        with zipfile.ZipFile(inner, "w", zipfile.ZIP_DEFLATED) as z:
            z.writestr("notes.txt", rng.choice(EMAIL_BODIES)[1])
        data = inner.getvalue()
    else:  # txt
        data = rng.choice(EMAIL_BODIES)[1].encode("utf-8")

    return filename, maintype, subtype, data


def _get_display_name(email: str) -> str:
    """Derives a display name from the email address."""
    local_part = email.split("@")[0]
    # Replace dots with spaces and capitalize each word
    name = local_part.replace(".", " ").title()
    # Handle some special cases if needed, but .title() covers most
    return name


def build_message(rng: random.Random, include_attachments: bool) -> EmailMessage:
    sender_email = rng.choice(EMAIL_ADDRESSES)
    recipient_email = rng.choice([a for a in EMAIL_ADDRESSES if a != sender_email])
    sender_name = _get_display_name(sender_email)
    recipient_name = _get_display_name(recipient_email)
    subject, body_text = rng.choice(EMAIL_BODIES)

    # Occasionally add a Re:/Fwd: prefix for realism
    prefix = rng.choice(["", "", "", "Re: ", "Fwd: "])
    if prefix and not subject.startswith(("Re:", "Fwd:")):
        subject = f"{prefix}{subject}"

    msg = EmailMessage()
    msg["Subject"] = subject
    msg["From"] = formataddr((sender_name, sender_email))
    msg["To"] = formataddr((recipient_name, recipient_email))
    msg["Date"] = formatdate(localtime=True)
    msg["Message-ID"] = make_msgid()

    signoff = rng.choice([
        "Transmission ends.",
        "End of log.",
        "Await further instructions.",
        "Stay vigilant.",
        "Godspeed.",
        "This channel will now go dark.",
    ])
    full_text = f"{body_text}\n\n{signoff}\n\n-- {sender_name} <{sender_email}>"

    timestamp = datetime.now(timezone.utc).isoformat()
    
    # Themes
    themes = [
        # 1. Classic Terminal (Original)
        f"""<html>
  <body style="font-family: monospace; background:#0a0a0a; color:#c9f7c9; padding:20px;">
    <h2 style="color:#8ef58e;">{subject}</h2>
    <p>{body_text}</p>
    <p style="font-style:italic; color:#6fbf6f;">{signoff}</p>
    <hr style="border-color:#2f4f2f;">
    <p style="font-size:12px; color:#5a8f5a;">Sent from {sender_name} ({sender_email}) &middot; {timestamp}</p>
  </body>
</html>""",
        # 2. Cyberpunk Neon
        f"""<html>
  <body style="font-family: 'Courier New', Courier, monospace; background:#050505; color:#f0f; padding:20px; border: 2px solid #0ff;">
    <h2 style="color:#0ff; text-transform: uppercase; letter-spacing: 2px;">{subject}</h2>
    <p style="color:#fff; background: rgba(255,0,255,0.1); padding: 10px;">{body_text}</p>
    <p style="font-weight:bold; color:#f0f;">{signoff}</p>
    <div style="font-size:10px; color:#0ff; margin-top:20px; border-top: 1px dashed #0ff; padding-top: 10px;">
      NODE: {uuid.uuid4().hex[:8].upper()} | SRC: {sender_email} | {timestamp}
    </div>
  </body>
</html>""",
        # 3. Blueprint / Tactical
        f"""<html>
  <body style="font-family: sans-serif; background:#001a33; color:#80ccff; padding:30px; border-left: 5px solid #0059b3;">
    <div style="font-size: 10px; color: #0059b3; margin-bottom: 10px;">CLASSIFIED // EYES ONLY</div>
    <h2 style="color:#ffffff; margin-top: 0;">{subject}</h2>
    <div style="background: rgba(128,204,255,0.05); padding: 20px; border: 1px solid #0059b3;">
      <p style="line-height: 1.6;">{body_text}</p>
    </div>
    <p style="font-family: monospace; color:#ffffff; margin-top: 20px;">{signoff}</p>
    <p style="font-size:11px; color:#0059b3;">Encrypted link established &middot; {timestamp}</p>
  </body>
</html>""",
        # 4. Industrial / Weyland-Yutani Style
        f"""<html>
  <body style="font-family: 'Verdana', sans-serif; background:#222; color:#ccc; padding:0; margin:0;">
    <div style="background:#f39c12; color:#000; padding:10px; font-weight:bold; text-align:center;">
      INTERNAL MEMO - AUTHORIZED PERSONNEL ONLY
    </div>
    <div style="padding:30px;">
      <h2 style="color:#f39c12; border-bottom: 2px solid #f39c12; padding-bottom: 5px;">{subject}</h2>
      <p style="color:#eee;">{body_text}</p>
      <p style="margin-top:40px; border-left: 3px solid #f39c12; padding-left: 10px;">{signoff}<br><small>{sender_name}</small></p>
      <div style="margin-top:50px; font-size:10px; color:#666;">
        Property of Weyland-Yutani Corp. &middot; Building Better Worlds &middot; {timestamp}
      </div>
    </div>
  </body>
</html>""",
        # 5. Retro Amber Terminal
        f"""<html>
  <body style="font-family: 'Lucida Console', Monaco, monospace; background:#1a1100; color:#ffb000; padding:20px;">
    <div style="border: 1px solid #ffb000; padding: 15px;">
      <div style="background:#ffb000; color:#1a1100; padding: 2px 10px; display:inline-block; font-weight:bold; margin-bottom:15px;">
        COMM-LINK ACTIVE
      </div>
      <h2 style="margin-top:0;">> {subject}</h2>
      <p>> {body_text}</p>
      <p style="margin-top:30px;">> {signoff}</p>
      <div style="margin-top:20px; font-size:12px; opacity:0.7;">
        [AUTHENTICATION VERIFIED] -- {timestamp}
      </div>
    </div>
  </body>
</html>""",
        # 6. Minimalist Space Station
        f"""<html>
  <body style="font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background:#f0f4f8; color:#2d3748; padding:40px;">
    <div style="max-width: 600px; margin: 0 auto; background: #ffffff; padding: 40px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
      <div style="color: #4a5568; font-size: 12px; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 20px; border-bottom: 1px solid #e2e8f0; padding-bottom: 10px;">
        Station Communication Hub
      </div>
      <h2 style="color:#1a202c; margin-top: 0;">{subject}</h2>
      <p style="font-size: 16px; line-height: 1.6;">{body_text}</p>
      <p style="color:#718096; font-style: italic; margin-top: 30px;">{signoff}</p>
      <div style="margin-top:40px; padding-top: 20px; border-top: 1px solid #e2e8f0; font-size: 12px; color: #a0aec0; text-align: center;">
        Orbit: Earth-Moon L1 &middot; {timestamp}
      </div>
    </div>
  </body>
</html>"""
    ]

    html_body = rng.choice(themes)

    msg.set_content(full_text)
    msg.add_alternative(html_body, subtype="html")

    if include_attachments and rng.random() < 0.7:
        num_attachments = rng.randint(1, 2)
        for _ in range(num_attachments):
            filename, maintype, subtype_, data = make_random_attachment(rng)
            msg.add_attachment(
                data, maintype=maintype, subtype=subtype_, filename=filename
            )

    return msg


def main():
    parser = argparse.ArgumentParser(
        description="Send realistic sci-fi-themed test emails to an SMTP server."
    )
    parser.add_argument("--host", default="localhost", help="SMTP server host (default: localhost)")
    parser.add_argument("--port", type=int, default=1025, help="SMTP server port (default: 1025)")
    parser.add_argument("--count", type=int, default=10, help="Number of emails to send (default: 10)")
    parser.add_argument(
        "--attachments", action="store_true",
        help="Randomly attach files (pdf, docx, txt, csv, zip, png) to some emails."
    )
    parser.add_argument("--seed", type=int, default=None, help="Random seed for reproducibility")
    parser.add_argument(
        "--dry-run", action="store_true",
        help="Build messages and print summaries without connecting to an SMTP server."
    )
    args = parser.parse_args()

    rng = random.Random(args.seed)

    smtp = None
    if not args.dry_run:
        smtp = smtplib.SMTP(args.host, args.port)

    try:
        for i in range(1, args.count + 1):
            msg = build_message(rng, include_attachments=args.attachments)
            attachment_count = sum(
                1 for part in msg.iter_attachments()
            )
            print(
                f"[{i}/{args.count}] From: {msg['From']} -> To: {msg['To']} "
                f"| Subject: {msg['Subject']} | Attachments: {attachment_count}"
            )
            if not args.dry_run:
                smtp.send_message(msg)
    finally:
        if smtp is not None:
            smtp.quit()

    print(f"\nDone. {'Simulated' if args.dry_run else 'Sent'} {args.count} email(s).")


if __name__ == "__main__":
    main()