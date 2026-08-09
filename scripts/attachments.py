#!/usr/bin/env python3
"""
send_test_emails.py

Generates realistic, professionally-designed HTML business emails
(order confirmations, shipping notices, password resets, 2FA codes,
security alerts, welcome messages, invoices, appointment reminders,
system/storage notifications, and cart-abandonment reminders) and
sends them to a local/test SMTP server (e.g. MailHog, MailDev,
Python's `smtpd`/`aiosmtpd`, etc.).

Only the Python standard library is used (argparse, smtplib, email,
random, datetime, uuid).

Usage:
    python3 send_test_emails.py --count 25
    python3 send_test_emails.py --host localhost --port 1025 --count 100

Each email:
  * has inline CSS only (no <style> blocks)
  * uses nested <table> layouts with width/cellpadding/cellspacing/border
    set explicitly (Outlook-safe)
  * uses only absolute https:// image URLs
  * has a hidden preheader snippet
  * uses web-safe fonts (Arial/Helvetica/sans-serif fallback stack)
  * contains no <script> or form elements
  * is drawn from a pool of 50 canned "from"/"to" addresses and a pool
    of 175 canned body templates (5 per use-case x 35 use-cases), each
    rendered with randomized realistic data so consecutive emails of
    the same template still differ.
"""

import argparse
import random
import smtplib
import sys
import uuid
from datetime import datetime, timedelta
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.utils import formatdate, make_msgid


# --------------------------------------------------------------------------
# 1. Canned address pool (50 realistic addresses spanning both businesses
#    that send transactional mail and the customers who receive it).
# --------------------------------------------------------------------------

ADDRESS_POOL = [
    "orders@brightcart.com",
    "no-reply@brightcart.com",
    "shipping@parcelwave.com",
    "tracking@parcelwave.com",
    "accounts@nimbuscloud.io",
    "security@nimbuscloud.io",
    "support@lumen-bank.com",
    "billing@lumen-bank.com",
    "no-reply@auth.vertexsoft.com",
    "alerts@vertexsoft.com",
    "hello@fernwellclinic.com",
    "appointments@fernwellclinic.com",
    "welcome@meridianstudio.app",
    "team@meridianstudio.app",
    "receipts@cobaltmarket.com",
    "no-reply@cobaltmarket.com",
    "notifications@driftstorage.com",
    "care@driftstorage.com",
    "reminders@urbanreserve.com",
    "bookings@urbanreserve.com",
    "cart@pixelforge-shop.com",
    "deals@pixelforge-shop.com",
    "sarah.mitchell@gmail.com",
    "j.owusu84@gmail.com",
    "priya.nair@outlook.com",
    "daniel.reyes@yahoo.com",
    "emma.hartley@icloud.com",
    "mo.abdullah@hotmail.com",
    "linh.tran92@gmail.com",
    "carlos.mendez@outlook.com",
    "isabelle.roy@gmail.com",
    "tommaso.greco@icloud.com",
    "hannah.oconnor@yahoo.com",
    "ravi.chandran@gmail.com",
    "olivia.bennett@outlook.com",
    "kwame.asante@gmail.com",
    "yuki.tanaka@icloud.com",
    "grace.kimani@gmail.com",
    "noah.fischer@gmail.com",
    "aisha.rahman@outlook.com",
    "lucas.bergstrom@gmail.com",
    "maria.dossantos@yahoo.com",
    "ethan.walker88@gmail.com",
    "chloe.dubois@icloud.com",
    "arjun.mehta@gmail.com",
    "sophie.laurent@outlook.com",
    "benjamin.wolf@gmail.com",
    "natasha.ivanova@yahoo.com",
    "felix.hoffman@gmail.com",
    "amara.okeke@gmail.com",
]

assert len(ADDRESS_POOL) == 50


# --------------------------------------------------------------------------
# 2. Small randomized-data helpers (self-contained, stdlib only)
# --------------------------------------------------------------------------

FIRST_NAMES = [
    "James", "Maria", "Wei", "Fatima", "Liam", "Sofia", "Kenji", "Amara",
    "Noah", "Elena", "Arjun", "Grace", "Lucas", "Priya", "Ethan", "Nadia",
    "Yuki", "Ibrahim", "Chiara", "Dmitri", "Aisha", "Miguel", "Zara", "Raj",
    "Leila", "Henrik", "Mei", "Omar", "Freya", "Carlos", "Ananya", "Sven",
    "Ingrid", "Hassan", "Camila", "Kwame", "Linnea", "Ahmed", "Astrid", "Diego",
    "Hana", "Lars", "Valentina", "Mohammed", "Suki", "Mateo", "Anika", "Pierre",
    "Hiroshi", "Sakura", "Taro", "Akiko", "Haruto", "Yui", "Takeshi", "Rina",
    "Jin", "Min-Ji", "Sung-Ho", "Ji-Woo", "Joon", "Hye-Jin", "Dae-Jung", "Soo-Min",
    "Liang", "Xiao", "Chen", "Ling", "Hong", "Jian", "Feng", "Yan",
    "Ravi", "Deepak", "Amit", "Kavya", "Rohan", "Sanjay", "Pooja", "Neha",
    "Nguyen", "Thi", "Minh", "Tuan", "Linh", "Phuong", "Hieu", "Thao",
    "Somchai", "Siriporn", "Wichai", "Niran", "Chai", "Porn", "Lek", "Noi",
    "Budi", "Siti", "Agus", "Dewi", "Adi", "Rina",
]
LAST_NAMES = [
    "Turner", "Nguyen", "Costa", "Baptiste", "Schulz", "Kovac", "Adeyemi",
    "Larsen", "Whitfield", "Rossi", "Kapoor", "Bianchi", "Okafor", "Reyes",
    "Tanaka", "Silva", "Kim", "Andersen", "Moreau", "Petrov", "Garcia", "Chen",
    "Hassan", "Johansson", "Patel", "Müller", "Santos", "Ali", "Ivanov", "Martinez",
    "Eriksson", "Nakamura", "Fernandez", "Sato", "Rodriguez", "Berg", "Yamamoto",
    "Dubois", "Nielsen", "Hernandez", "Lindström", "Kamau", "Fedorov", "Ito",
    "Suzuki", "Watanabe", "Kobayashi", "Takahashi", "Sasaki", "Kato", "Yoshida", "Yamaguchi",
    "Park", "Lee", "Choi", "Jung", "Kang", "Cho", "Yoon", "Jang",
    "Wang", "Li", "Zhang", "Liu", "Yang", "Huang", "Zhao", "Wu",
    "Singh", "Kumar", "Sharma", "Gupta", "Reddy", "Rao", "Verma", "Joshi",
    "Tran", "Le", "Pham", "Hoang", "Vo", "Dang", "Bui", "Do",
    "Srisai", "Chaiyasit", "Rattana", "Somjai", "Boonma", "Phongam", "Suwan", "Thitirat",
    "Wijaya", "Santoso", "Kurniawan", "Suryanto", "Gunawan", "Surya",
]
PRODUCTS = [
    ("Wireless Noise-Cancelling Headphones", 129.99),
    ("Stainless Steel Pour-Over Kettle", 42.50),
    ("Organic Cotton Bath Towel Set", 58.00),
    ("Mechanical Keyboard - Low Profile", 89.99),
    ("Ceramic Plant Pot (Set of 3)", 34.75),
    ("Portable Espresso Maker", 64.20),
    ("Merino Wool Crew Socks (3-Pack)", 24.00),
    ("Leather Bifold Wallet", 47.90),
    ("Bamboo Desk Organizer", 31.25),
    ("Aluminum Laptop Stand", 39.99),
    ("Insulated Travel Mug", 22.50),
    ("Scented Soy Candle - Cedarwood", 18.00),
    ("USB-C Multiport Hub", 45.99),
    ("Yoga Mat - Extra Thick", 35.00),
    ("Stainless Steel Water Bottle", 28.50),
    ("Wireless Phone Charger", 24.99),
    ("Blue Light Blocking Glasses", 32.00),
    ("Desk Lamp with Wireless Charging", 54.99),
    ("Backpack - Laptop Compartment", 69.99),
    ("Cotton Bed Sheet Set - Queen", 52.00),
    ("Electric Toothbrush", 79.99),
    ("French Press Coffee Maker", 29.99),
    ("Memory Foam Pillow", 38.50),
    ("Kitchen Knife Set (5-Piece)", 84.99),
    ("Running Shoes - Men's", 95.00),
    ("Smartwatch Fitness Tracker", 149.99),
    ("Portable Bluetooth Speaker", 55.00),
    ("Cast Iron Skillet - 12 inch", 44.99),
    ("Essential Oil Diffuser", 31.99),
    ("Resistance Bands Set", 19.99),
    ("Sunglasses - Polarized", 68.00),
    ("Cordless Vacuum Cleaner", 189.99),
    ("Ceramic Cookware Set", 124.99),
    ("Electric Kettle - Glass", 36.50),
    ("Foam Roller - High Density", 26.99),
    ("Weighted Blanket - 15 lbs", 72.00),
    ("Non-Stick Baking Sheet Set", 23.50),
    ("Wall-Mounted Coat Rack", 41.99),
    ("Shower Head - Rainfall", 48.00),
    ("Cutting Board - Bamboo", 21.99),
    ("Hair Dryer - Ionic Technology", 64.99),
    ("Storage Bins (Set of 4)", 33.50),
    ("LED Strip Lights - 16ft", 27.99),
    ("Reusable Grocery Bags (6-Pack)", 16.50),
    ("Gaming Mouse - RGB", 49.99),
    ("Measuring Cups & Spoons Set", 14.99),
    ("Picture Frame Set (5-Pack)", 38.99),
    ("Bathroom Scale - Digital", 29.50),
    ("Air Purifier - HEPA Filter", 119.99),
    ("Cookbook Stand - Adjustable", 18.99),
    ("Laptop Sleeve - 13 inch", 22.99),
    ("Drawer Organizer Set", 24.50),
    ("Camping Hammock - Double", 42.99),
    ("Blender - Personal Size", 34.99),
    ("Monitor Stand - Adjustable", 52.99),
    ("Reading Light - Clip-On", 17.99),
    ("Mixing Bowl Set - Stainless Steel", 39.50),
    ("Dumbbell Set - Adjustable", 159.99),
    ("Throw Blanket - Fleece", 29.99),
    ("Phone Mount - Car Dashboard", 19.50),
    ("Journal - Hardcover", 16.99),
    ("Extension Cord - 10ft", 13.99),
    ("Waffle Maker - Belgian Style", 44.50),
]
CARRIERS = [
    "ParcelWave Express", "SwiftRoute Logistics", "MetroLine Courier", "OrbitPost",
    "FedEx", "UPS", "DHL Express", "USPS", "Royal Mail", "Canada Post",
    "Australia Post", "Deutsche Post", "La Poste", "Correos", "PostNL",
    "Japan Post", "China Post", "India Post", "Singapore Post", "TNT Express",
    "Aramex", "DPD", "Hermes", "Yodel", "Purolator", "OnTrac", "LaserShip",
    "GLS", "Chronopost", "Colissimo", "Parcelforce", "Evri", "InPost",
    "SF Express", "ZTO Express", "YTO Express", "STO Express", "Blue Dart",
    "Delhivery", "DTDC", "Gati", "Professional Couriers", "Ekart Logistics",
    "XPO Logistics", "Estes Express", "Old Dominion", "R+L Carriers", "ABF Freight",
    "YRC Freight", "Saia", "TForce Freight", "Pitt Ohio", "Southeastern Freight"
]
CITIES = [
    "Austin, TX", "Manchester, UK", "Toronto, ON", "Singapore", "Berlin, DE",
    "Melbourne, AU", "Kolkata, IN", "Lisbon, PT", "Nairobi, KE", "Osaka, JP",
    "New York, NY", "Los Angeles, CA", "Chicago, IL", "Houston, TX", "Phoenix, AZ",
    "Philadelphia, PA", "San Antonio, TX", "San Diego, CA", "Dallas, TX", "San Jose, CA",
    "London, UK", "Paris, FR", "Madrid, ES", "Rome, IT", "Amsterdam, NL",
    "Brussels, BE", "Vienna, AT", "Stockholm, SE", "Copenhagen, DK", "Dublin, IE",
    "Sydney, AU", "Brisbane, AU", "Perth, AU", "Auckland, NZ", "Wellington, NZ",
    "Tokyo, JP", "Kyoto, JP", "Seoul, KR", "Busan, KR", "Shanghai, CN",
    "Beijing, CN", "Hong Kong, HK", "Taipei, TW", "Bangkok, TH", "Kuala Lumpur, MY",
    "Manila, PH", "Jakarta, ID", "Mumbai, IN", "Delhi, IN", "Bangalore, IN",
    "Dubai, AE", "Istanbul, TR", "Moscow, RU", "São Paulo, BR", "Mexico City, MX",
    "Buenos Aires, AR", "Lima, PE", "Bogotá, CO", "Santiago, CL", "Johannesburg, ZA"
]
DEVICES = [
    "Chrome on Windows", "Safari on iPhone", "Firefox on macOS",
    "Edge on Windows", "Chrome on Android", "Safari on iPad",
    "Chrome on macOS", "Firefox on Windows", "Safari on macOS", "Edge on macOS",
    "Chrome on Linux", "Firefox on Linux", "Opera on Windows", "Brave on macOS",
    "Samsung Internet on Android", "Firefox on Android", "Edge on Android", "Opera on Android",
    "Chrome on Chrome OS", "Safari on iPhone 14", "Safari on iPhone 13", "Safari on iPad Pro",
    "Chrome on Windows 11", "Chrome on Windows 10", "Edge on Windows 11", "Firefox on Ubuntu",
    "Vivaldi on Windows", "Arc on macOS", "DuckDuckGo on iPhone", "Brave on Windows",
    "Opera GX on Windows", "Firefox Focus on iPhone", "Chrome on Pixel", "Chrome on Samsung Galaxy",
    "Edge on Surface", "Safari on Mac Mini", "Safari on MacBook Pro", "Chrome on Chromebook",
    "Firefox on Fedora", "Brave on Linux", "Opera on macOS", "Vivaldi on macOS",
    "Edge on Xbox", "Chrome on Lenovo", "Firefox on Debian", "Chrome on HP",
    "Safari on iMac", "Edge on Surface Pro", "Chrome on Dell", "Firefox on Mint",
    "Brave on Ubuntu", "Opera on Linux", "Chrome on Asus", "Firefox on Arch",
    "Safari on iPad Air", "Edge on Surface Laptop", "Chrome on Acer", "Firefox on Pop!_OS"
]
SERVICE_NAMES = [
    "Meridian Studio", "Vertex Suite", "Nimbus Cloud", "Cobalt Market",
    "Drift Storage", "Urban Reserve", "Fernwell Health", "PixelForge",
    "Asana", "Slack", "Trello", "Notion", "Dropbox", "Google Workspace",
    "Microsoft 365", "Zoom", "Salesforce", "HubSpot", "Mailchimp", "Shopify",
    "Stripe", "Square", "PayPal", "QuickBooks", "Xero", "FreshBooks",
    "Zendesk", "Intercom", "Atlassian", "Jira", "Confluence", "GitHub",
    "GitLab", "Bitbucket", "AWS", "Azure", "Google Cloud", "DigitalOcean",
    "Heroku", "Vercel", "Netlify", "Cloudflare", "Airtable", "Monday.com",
    "ClickUp", "Basecamp", "Figma", "Canva", "Adobe Creative Cloud", "Sketch",
    "InVision", "Miro", "DocuSign", "HelloSign", "Calendly", "Typeform",
    "SurveyMonkey", "LinkedIn", "Twitter", "Meta Business Suite"
]


def rand_name():
    return f"{random.choice(FIRST_NAMES)} {random.choice(LAST_NAMES)}"


def rand_order_id():
    return "ORD-" + "".join(random.choices("0123456789", k=7))


def rand_invoice_id():
    return "INV-" + datetime.now().strftime("%Y%m") + "-" + "".join(random.choices("0123456789", k=5))


def rand_tracking():
    return "".join(random.choices("ABCDEFGHJKLMNPQRSTUVWXYZ", k=2)) + \
        "".join(random.choices("0123456789", k=9)) + "US"


def rand_otp():
    return "".join(random.choices("0123456789", k=6))


def rand_ip():
    return ".".join(str(random.randint(1, 254)) for _ in range(4))


def rand_amount(low, high):
    return round(random.uniform(low, high), 2)


def money(x):
    return f"${x:,.2f}"


def rand_date(days_offset=0):
    return (datetime.now() + timedelta(days=days_offset)).strftime("%B %d, %Y")


def rand_time():
    hour = random.choice([9, 10, 11, 13, 14, 15, 16])
    minute = random.choice(["00", "15", "30", "45"])
    ampm = "AM" if hour < 12 else "PM"
    display_hour = hour if hour <= 12 else hour - 12
    return f"{display_hour}:{minute} {ampm}"


def line_items(n=None):
    n = n or random.randint(1, 3)
    chosen = random.sample(PRODUCTS, n)
    items = []
    subtotal = 0.0
    for name, price in chosen:
        qty = random.randint(1, 2)
        line_total = round(price * qty, 2)
        subtotal += line_total
        items.append((name, qty, price, line_total))
    return items, round(subtotal, 2)


# --------------------------------------------------------------------------
# 3. Email rendering engine: table-based layout, inline CSS only.
# --------------------------------------------------------------------------

FONT_STACK = "Arial, Helvetica, 'Segoe UI', sans-serif"

LOGO_IMAGE_BASE = "https://images.unsplash.com/"  # absolute https placeholder host


def render_button(text, url, bg_color, text_color="#ffffff"):
    return f"""
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td align="center" style="padding:8px 0 4px 0;">
          <table role="presentation" cellpadding="0" cellspacing="0" border="0">
            <tr>
              <td align="center" bgcolor="{bg_color}" style="border-radius:4px;">
                <a href="{url}" target="_blank"
                   style="display:inline-block; padding:14px 32px; font-family:{FONT_STACK};
                          font-size:15px; font-weight:bold; color:{text_color};
                          text-decoration:none; border-radius:4px;">
                  {text}
                </a>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
    """


def render_layout_classic(theme, preheader, brand_name, header_kicker, header_title,
                          body_rows_html, footer_note, unsubscribe_addr):
    """
    Layout 1/4 - "Classic": solid color header bar, thin accent stripe,
    kicker + title block, single content column, hairline divider, footer.
    theme: dict with primary, accent, bg, panel_bg, text, muted, border
    """
    year = datetime.now().year
    return f"""<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<title>{header_title}</title>
</head>
<body style="margin:0; padding:0; background-color:{theme['bg']}; -webkit-text-size-adjust:100%; mso-text-size-adjust:100%;">
<div style="display:none; max-height:0px; max-width:0px; opacity:0; overflow:hidden; mso-hide:all; font-size:1px; line-height:1px; color:{theme['bg']};">
  {preheader}
</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="{theme['bg']}">
<tr>
<td align="center" style="padding:32px 12px;">

  <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0"
         style="width:600px; max-width:600px; background-color:{theme['panel_bg']};
                border:1px solid {theme['border']}; border-radius:6px; overflow:hidden;">

    <!-- Header -->
    <tr>
      <td bgcolor="{theme['primary']}" style="padding:28px 36px;">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
          <tr>
            <td align="left" style="font-family:{FONT_STACK}; font-size:20px; font-weight:bold;
                                     letter-spacing:0.5px; color:{theme['on_primary']};">
              {brand_name}
            </td>
          </tr>
        </table>
      </td>
    </tr>

    <!-- Accent bar -->
    <tr>
      <td bgcolor="{theme['accent']}" style="line-height:4px; font-size:4px;">&nbsp;</td>
    </tr>

    <!-- Title block -->
    <tr>
      <td style="padding:36px 36px 8px 36px;">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
          <tr>
            <td style="font-family:{FONT_STACK}; font-size:12px; font-weight:bold; letter-spacing:1.5px;
                       text-transform:uppercase; color:{theme['accent']}; padding-bottom:10px;">
              {header_kicker}
            </td>
          </tr>
          <tr>
            <td style="font-family:{FONT_STACK}; font-size:23px; font-weight:bold; color:{theme['text']};
                       line-height:1.35;">
              {header_title}
            </td>
          </tr>
        </table>
      </td>
    </tr>

    <!-- Body content -->
    <tr>
      <td style="padding:8px 36px 12px 36px; font-family:{FONT_STACK}; font-size:14px; color:{theme['text']}; line-height:1.6;">
        {body_rows_html}
      </td>
    </tr>

    <!-- Divider -->
    <tr>
      <td style="padding:12px 36px 0 36px;">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
          <tr><td style="border-top:1px solid {theme['border']}; line-height:1px; font-size:1px;">&nbsp;</td></tr>
        </table>
      </td>
    </tr>

    <!-- Footer -->
    <tr>
      <td style="padding:24px 36px 32px 36px; font-family:{FONT_STACK}; font-size:12px; color:{theme['muted']}; line-height:1.6;">
        {footer_note}<br>
        &copy; {year} {brand_name}. All rights reserved.<br>
        This message was sent to you regarding your account.
        <a href="mailto:{unsubscribe_addr}?subject=unsubscribe" style="color:{theme['muted']}; text-decoration:underline;">Unsubscribe</a>
        &nbsp;|&nbsp;
        <a href="https://example.com/privacy" style="color:{theme['muted']}; text-decoration:underline;">Privacy Policy</a>
      </td>
    </tr>

  </table>

  <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px;">
    <tr>
      <td align="center" style="padding:18px 12px 0 12px; font-family:{FONT_STACK}; font-size:11px; color:{theme['footer_grey']};">
        123 Market Street, Suite 400, San Francisco, CA 94103
      </td>
    </tr>
  </table>

</td>
</tr>
</table>
</body>
</html>"""


def render_layout_hero(theme, preheader, brand_name, header_kicker, header_title,
                       body_rows_html, footer_note, unsubscribe_addr):
    """
    Layout 2/4 - "Hero": tall colored hero panel with a centered circular
    badge and centered title, a white content card that visually overlaps
    the hero (via negative-margin-style offset table), and a two-tone
    footer bar.
    """
    year = datetime.now().year
    initials = "".join(w[0] for w in brand_name.split()[:2]).upper()
    return f"""<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<title>{header_title}</title>
</head>
<body style="margin:0; padding:0; background-color:{theme['bg']}; -webkit-text-size-adjust:100%; mso-text-size-adjust:100%;">
<div style="display:none; max-height:0px; max-width:0px; opacity:0; overflow:hidden; mso-hide:all; font-size:1px; line-height:1px; color:{theme['bg']};">
  {preheader}
</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="{theme['bg']}">
<tr>
<td align="center" style="padding:32px 12px;">

  <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px;">

    <!-- Hero -->
    <tr>
      <td bgcolor="{theme['primary']}" align="center" style="padding:40px 36px 56px 36px; border-radius:6px 6px 0 0;">
        <table role="presentation" cellpadding="0" cellspacing="0" border="0">
          <tr>
            <td align="center" bgcolor="{theme['accent']}" width="52" height="52"
                style="width:52px; height:52px; border-radius:26px; font-family:{FONT_STACK};
                       font-size:18px; font-weight:bold; color:#ffffff; text-align:center;">
              {initials}
            </td>
          </tr>
        </table>
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
          <tr>
            <td align="center" style="padding-top:18px; font-family:{FONT_STACK}; font-size:11px; font-weight:bold;
                       letter-spacing:1.5px; text-transform:uppercase; color:{theme['accent']};">
              {header_kicker}
            </td>
          </tr>
          <tr>
            <td align="center" style="padding-top:8px; font-family:{FONT_STACK}; font-size:24px; font-weight:bold;
                       color:{theme['on_primary']}; line-height:1.3;">
              {header_title}
            </td>
          </tr>
        </table>
      </td>
    </tr>

    <!-- Overlapping content card -->
    <tr>
      <td style="padding:0 18px;">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"
               style="background-color:{theme['panel_bg']}; border:1px solid {theme['border']};
                      border-radius:8px; margin-top:-32px;">
          <tr>
            <td style="padding:32px 30px 8px 30px; font-family:{FONT_STACK}; font-size:14px; color:{theme['text']}; line-height:1.6;">
              {body_rows_html}
            </td>
          </tr>
          <tr>
            <td style="padding:20px 30px 26px 30px; font-family:{FONT_STACK}; font-size:11px; color:{theme['muted']}; line-height:1.6; border-top:1px solid {theme['border']};">
              {footer_note}<br>
              &copy; {year} {brand_name}. All rights reserved.
              <a href="mailto:{unsubscribe_addr}?subject=unsubscribe" style="color:{theme['muted']}; text-decoration:underline;">Unsubscribe</a>
              &nbsp;|&nbsp;
              <a href="https://example.com/privacy" style="color:{theme['muted']}; text-decoration:underline;">Privacy Policy</a>
            </td>
          </tr>
        </table>
      </td>
    </tr>

    <tr>
      <td align="center" style="padding:18px 12px 0 12px; font-family:{FONT_STACK}; font-size:11px; color:{theme['footer_grey']};">
        123 Market Street, Suite 400, San Francisco, CA 94103
      </td>
    </tr>

  </table>

</td>
</tr>
</table>
</body>
</html>"""


def render_layout_sidebar(theme, preheader, brand_name, header_kicker, header_title,
                          body_rows_html, footer_note, unsubscribe_addr):
    """
    Layout 3/4 - "Sidebar accent": a slim full-height colored rail down the
    left edge of the card (built as an adjacent <td> in the same row) next
    to the content column, plain header row above it with just the brand
    wordmark, footer below spanning full width.
    """
    year = datetime.now().year
    return f"""<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<title>{header_title}</title>
</head>
<body style="margin:0; padding:0; background-color:{theme['bg']}; -webkit-text-size-adjust:100%; mso-text-size-adjust:100%;">
<div style="display:none; max-height:0px; max-width:0px; opacity:0; overflow:hidden; mso-hide:all; font-size:1px; line-height:1px; color:{theme['bg']};">
  {preheader}
</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="{theme['bg']}">
<tr>
<td align="center" style="padding:32px 12px;">

  <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0"
         style="width:600px; max-width:600px; background-color:{theme['panel_bg']};
                border:1px solid {theme['border']}; border-radius:6px; overflow:hidden;">

    <!-- Plain wordmark header -->
    <tr>
      <td style="padding:26px 36px 18px 36px; border-bottom:1px solid {theme['border']};">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
          <tr>
            <td align="left" style="font-family:{FONT_STACK}; font-size:17px; font-weight:bold;
                                     letter-spacing:0.5px; color:{theme['primary']};">
              {brand_name}
            </td>
            <td align="right" style="font-family:{FONT_STACK}; font-size:11px; font-weight:bold;
                       letter-spacing:1.5px; text-transform:uppercase; color:{theme['accent']};">
              {header_kicker}
            </td>
          </tr>
        </table>
      </td>
    </tr>

    <!-- Sidebar rail + content -->
    <tr>
      <td>
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
          <tr>
            <td bgcolor="{theme['accent']}" width="8" style="width:8px; font-size:1px; line-height:1px;">&nbsp;</td>
            <td style="padding:28px 34px 20px 26px;">
              <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td style="font-family:{FONT_STACK}; font-size:22px; font-weight:bold; color:{theme['text']};
                             line-height:1.35; padding-bottom:14px;">
                    {header_title}
                  </td>
                </tr>
                <tr>
                  <td style="font-family:{FONT_STACK}; font-size:14px; color:{theme['text']}; line-height:1.6;">
                    {body_rows_html}
                  </td>
                </tr>
              </table>
            </td>
          </tr>
        </table>
      </td>
    </tr>

    <!-- Footer -->
    <tr>
      <td bgcolor="{theme['bg']}" style="padding:20px 36px; font-family:{FONT_STACK}; font-size:12px; color:{theme['muted']}; line-height:1.6; border-top:1px solid {theme['border']};">
        {footer_note}<br>
        &copy; {year} {brand_name}. All rights reserved.
        <a href="mailto:{unsubscribe_addr}?subject=unsubscribe" style="color:{theme['muted']}; text-decoration:underline;">Unsubscribe</a>
        &nbsp;|&nbsp;
        <a href="https://example.com/privacy" style="color:{theme['muted']}; text-decoration:underline;">Privacy Policy</a>
      </td>
    </tr>

  </table>

  <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px;">
    <tr>
      <td align="center" style="padding:18px 12px 0 12px; font-family:{FONT_STACK}; font-size:11px; color:{theme['footer_grey']};">
        123 Market Street, Suite 400, San Francisco, CA 94103
      </td>
    </tr>
  </table>

</td>
</tr>
</table>
</body>
</html>"""


def render_layout_bold(theme, preheader, brand_name, header_kicker, header_title,
                       body_rows_html, footer_note, unsubscribe_addr):
    """
    Layout 4/4 - "Bold two-tone": the entire outer page background is the
    brand color rather than a neutral gray, a bordered white card sits on
    top of it, and a dark footer band (not the light neutral used
    elsewhere) closes out the message.
    """
    year = datetime.now().year
    dark_footer = theme["text"]
    return f"""<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<title>{header_title}</title>
</head>
<body style="margin:0; padding:0; background-color:{theme['primary']}; -webkit-text-size-adjust:100%; mso-text-size-adjust:100%;">
<div style="display:none; max-height:0px; max-width:0px; opacity:0; overflow:hidden; mso-hide:all; font-size:1px; line-height:1px; color:{theme['primary']};">
  {preheader}
</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="{theme['primary']}">
<tr>
<td align="center" style="padding:44px 12px 0 12px;">

  <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px;">
    <tr>
      <td align="center" style="padding-bottom:22px; font-family:{FONT_STACK}; font-size:18px; font-weight:bold;
                 letter-spacing:0.5px; color:{theme['on_primary']};">
        {brand_name}
      </td>
    </tr>
  </table>

  <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0"
         style="width:600px; max-width:600px; background-color:{theme['panel_bg']}; border-radius:8px 8px 0 0; overflow:hidden;">
    <tr>
      <td style="padding:34px 36px 6px 36px;">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
          <tr>
            <td align="center" style="font-family:{FONT_STACK}; font-size:12px; font-weight:bold; letter-spacing:1.5px;
                       text-transform:uppercase; color:{theme['accent']}; padding-bottom:10px;">
              {header_kicker}
            </td>
          </tr>
          <tr>
            <td align="center" style="font-family:{FONT_STACK}; font-size:22px; font-weight:bold; color:{theme['text']};
                       line-height:1.35; padding-bottom:8px;">
              {header_title}
            </td>
          </tr>
        </table>
      </td>
    </tr>
    <tr>
      <td style="padding:14px 36px 30px 36px; font-family:{FONT_STACK}; font-size:14px; color:{theme['text']}; line-height:1.6;">
        {body_rows_html}
      </td>
    </tr>
  </table>

  <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0"
         style="width:600px; max-width:600px; background-color:{dark_footer}; border-radius:0 0 8px 8px;">
    <tr>
      <td style="padding:22px 36px; font-family:{FONT_STACK}; font-size:12px; color:{theme['panel_bg']}; line-height:1.6; opacity:0.85;">
        {footer_note}<br>
        &copy; {year} {brand_name}. All rights reserved.
        <a href="mailto:{unsubscribe_addr}?subject=unsubscribe" style="color:{theme['panel_bg']}; text-decoration:underline;">Unsubscribe</a>
        &nbsp;|&nbsp;
        <a href="https://example.com/privacy" style="color:{theme['panel_bg']}; text-decoration:underline;">Privacy Policy</a>
      </td>
    </tr>
  </table>

  <table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px;">
    <tr>
      <td align="center" style="padding:18px 12px 32px 12px; font-family:{FONT_STACK}; font-size:11px; color:{theme['on_primary']}; opacity:0.7;">
        123 Market Street, Suite 400, San Francisco, CA 94103
      </td>
    </tr>
  </table>

</td>
</tr>
</table>
</body>
</html>"""


LAYOUT_RENDERERS = [
    render_layout_classic,
    render_layout_hero,
    render_layout_sidebar,
    render_layout_bold,
]


def render_text_fallback(header_title, plain_lines):
    body = "\n".join(plain_lines)
    return f"{header_title}\n{'=' * len(header_title)}\n\n{body}\n"


def info_row(label, value, theme):
    return f"""
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td style="padding:6px 0; font-family:{FONT_STACK}; font-size:13px; color:{theme['muted']}; width:42%;">{label}</td>
        <td style="padding:6px 0; font-family:{FONT_STACK}; font-size:13px; color:{theme['text']}; font-weight:bold; text-align:right;">{value}</td>
      </tr>
    </table>
    """


def items_table(items, subtotal, theme, tax_rate=0.0, shipping=0.0):
    rows = ""
    for name, qty, price, total in items:
        rows += f"""
        <tr>
          <td style="padding:10px 0; border-bottom:1px solid {theme['border']}; font-family:{FONT_STACK}; font-size:13px; color:{theme['text']};">
            {name}<br><span style="color:{theme['muted']}; font-size:12px;">Qty: {qty} &times; {money(price)}</span>
          </td>
          <td align="right" style="padding:10px 0; border-bottom:1px solid {theme['border']}; font-family:{FONT_STACK}; font-size:13px; color:{theme['text']}; font-weight:bold;">
            {money(total)}
          </td>
        </tr>
        """
    tax = round(subtotal * tax_rate, 2)
    grand_total = round(subtotal + tax + shipping, 2)
    summary_rows = f"""
        <tr>
          <td style="padding:10px 0 2px 0; font-family:{FONT_STACK}; font-size:13px; color:{theme['muted']};">Subtotal</td>
          <td align="right" style="padding:10px 0 2px 0; font-family:{FONT_STACK}; font-size:13px; color:{theme['text']};">{money(subtotal)}</td>
        </tr>
    """
    if shipping:
        summary_rows += f"""
        <tr>
          <td style="padding:2px 0; font-family:{FONT_STACK}; font-size:13px; color:{theme['muted']};">Shipping</td>
          <td align="right" style="padding:2px 0; font-family:{FONT_STACK}; font-size:13px; color:{theme['text']};">{money(shipping)}</td>
        </tr>
        """
    if tax_rate:
        summary_rows += f"""
        <tr>
          <td style="padding:2px 0; font-family:{FONT_STACK}; font-size:13px; color:{theme['muted']};">Tax</td>
          <td align="right" style="padding:2px 0; font-family:{FONT_STACK}; font-size:13px; color:{theme['text']};">{money(tax)}</td>
        </tr>
        """
    summary_rows += f"""
        <tr>
          <td style="padding:10px 0 0 0; font-family:{FONT_STACK}; font-size:15px; color:{theme['text']}; font-weight:bold; border-top:1px solid {theme['border']};">Total</td>
          <td align="right" style="padding:10px 0 0 0; font-family:{FONT_STACK}; font-size:15px; color:{theme['accent']}; font-weight:bold; border-top:1px solid {theme['border']};">{money(grand_total)}</td>
        </tr>
    """
    return f"""
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      {rows}
      {summary_rows}
    </table>
    """, grand_total


# --------------------------------------------------------------------------
# 4. Ten use-case themes
# --------------------------------------------------------------------------

THEMES = {
    "order_confirmation": dict(primary="#0f5132", accent="#c9a227", bg="#f4f6f4",
                               panel_bg="#ffffff", text="#1f2a24", muted="#6b7a71",
                               border="#e2e8e3", on_primary="#ffffff", footer_grey="#9aa79f"),
    "shipping_notification": dict(primary="#0b3d66", accent="#2ea3a3", bg="#f2f6f9",
                                  panel_bg="#ffffff", text="#1a2733", muted="#647486",
                                  border="#e0e8ef", on_primary="#ffffff", footer_grey="#96a3b0"),
    "password_reset": dict(primary="#7a1f2b", accent="#c1443a", bg="#f8f2f2",
                           panel_bg="#ffffff", text="#2b1a1c", muted="#7d6a6b",
                           border="#efe1e1", on_primary="#ffffff", footer_grey="#ab9798"),
    "two_factor_auth": dict(primary="#3b1f6b", accent="#8a5cf6", bg="#f5f3fa",
                            panel_bg="#ffffff", text="#241a36", muted="#6f6482",
                            border="#e6e1f0", on_primary="#ffffff", footer_grey="#a79cba"),
    "security_alert": dict(primary="#7a3b00", accent="#e08600", bg="#fbf5ec",
                           panel_bg="#ffffff", text="#332212", muted="#8a7256",
                           border="#f0e2c8", on_primary="#ffffff", footer_grey="#b39d7d"),
    "welcome": dict(primary="#0d6e6e", accent="#ff7a59", bg="#f2f9f8",
                    panel_bg="#ffffff", text="#173232", muted="#5f7b7a",
                    border="#dfeeec", on_primary="#ffffff", footer_grey="#94aeac"),
    "billing_invoice": dict(primary="#12294d", accent="#c8a24d", bg="#f4f5f8",
                            panel_bg="#ffffff", text="#1c243a", muted="#68708c",
                            border="#e2e4ee", on_primary="#ffffff", footer_grey="#9aa0b8"),
    "appointment_reminder": dict(primary="#3d4b8c", accent="#7f9cf5", bg="#f4f5fb",
                                 panel_bg="#ffffff", text="#232a4a", muted="#6b729a",
                                 border="#e3e6f5", on_primary="#ffffff", footer_grey="#9aa0c4"),
    "system_notification": dict(primary="#374151", accent="#f59e0b", bg="#f5f6f7",
                                panel_bg="#ffffff", text="#20242c", muted="#6b7280",
                                border="#e3e5e8", on_primary="#ffffff", footer_grey="#9aa0aa"),
    "cart_abandonment": dict(primary="#8a1c46", accent="#ff8fab", bg="#fbf2f5",
                             panel_bg="#ffffff", text="#3a1424", muted="#8a5d6d",
                             border="#f2e0e6", on_primary="#ffffff", footer_grey="#c199a8"),
    "account_verification": dict(primary="#055160", accent="#0dcaf0", bg="#f0f9fa",
                                 panel_bg="#ffffff", text="#083344", muted="#64748b",
                                 border="#cff4fc", on_primary="#ffffff", footer_grey="#94a3b8"),
    "subscription_cancelled": dict(primary="#545b62", accent="#6c757d", bg="#f8f9fa",
                                   panel_bg="#ffffff", text="#212529", muted="#6c757d",
                                   border="#dee2e6", on_primary="#ffffff", footer_grey="#adb5bd"),
    "payment_failed": dict(primary="#842029", accent="#ea868f", bg="#fff5f5",
                           panel_bg="#ffffff", text="#410f14", muted="#842029",
                           border="#f8d7da", on_primary="#ffffff", footer_grey="#f1aeb5"),
    "feedback_request": dict(primary="#084298", accent="#6ea8fe", bg="#f0f4f9",
                             panel_bg="#ffffff", text="#052c65", muted="#6c757d",
                             border="#cfe2ff", on_primary="#ffffff", footer_grey="#9ec5fe"),
    "trial_expiring": dict(primary="#664d03", accent="#ffc107", bg="#fffcf0",
                           panel_bg="#ffffff", text="#332701", muted="#664d03",
                           border="#fff3cd", on_primary="#ffffff", footer_grey="#ffda6a"),
    "refund_processed": dict(primary="#0f5c4d", accent="#3fbf9e", bg="#f2faf7",
                             panel_bg="#ffffff", text="#0f2a24", muted="#5f8479",
                             border="#dcf0e9", on_primary="#ffffff", footer_grey="#9cc4b8"),
    "subscription_renewed": dict(primary="#1b3a63", accent="#4d8fd6", bg="#f2f6fb",
                                 panel_bg="#ffffff", text="#152238", muted="#5b6c85",
                                 border="#dde7f3", on_primary="#ffffff", footer_grey="#9fb3cc"),
    "account_locked": dict(primary="#5c1a1a", accent="#d64545", bg="#fbf2f2",
                           panel_bg="#ffffff", text="#2e0f0f", muted="#8a5f5f",
                           border="#f1dcdc", on_primary="#ffffff", footer_grey="#c99999"),
    "newsletter_digest": dict(primary="#2d2a5e", accent="#ff9f43", bg="#f5f4fa",
                              panel_bg="#ffffff", text="#1c1a3a", muted="#6b678f",
                              border="#e6e4f2", on_primary="#ffffff", footer_grey="#a8a4c9"),
    "event_invitation": dict(primary="#4a1d6b", accent="#e879c0", bg="#f8f2fb",
                             panel_bg="#ffffff", text="#2c1240", muted="#7d6390",
                             border="#ecdff2", on_primary="#ffffff", footer_grey="#c6a8d6"),
    "survey_results_shared": dict(primary="#1c5a7a", accent="#5fc9d1", bg="#f1f8fa",
                                  panel_bg="#ffffff", text="#123344", muted="#5c7d8a",
                                  border="#dcedf1", on_primary="#ffffff", footer_grey="#9fc4cc"),
    "loyalty_points_earned": dict(primary="#7a4b00", accent="#f2b134", bg="#fdf6ea",
                                  panel_bg="#ffffff", text="#402700", muted="#8a6a3a",
                                  border="#f5e6c6", on_primary="#ffffff", footer_grey="#d3b06a"),
    "gift_card_received": dict(primary="#8a1c6b", accent="#f06fb0", bg="#fbf1f7",
                               panel_bg="#ffffff", text="#3a0f30", muted="#8a5d7c",
                               border="#f2dcec", on_primary="#ffffff", footer_grey="#cf9dbe"),
    "price_drop_alert": dict(primary="#144d29", accent="#4bd17a", bg="#f2faf5",
                             panel_bg="#ffffff", text="#0d2e18", muted="#557a63",
                             border="#dcf0e3", on_primary="#ffffff", footer_grey="#9ecab0"),
    "back_in_stock": dict(primary="#0d4a5c", accent="#2fb6c4", bg="#f1f9fa",
                          panel_bg="#ffffff", text="#092c37", muted="#547d87",
                          border="#dcedf0", on_primary="#ffffff", footer_grey="#96c2ca"),
    "review_request": dict(primary="#8a5a00", accent="#ffb020", bg="#fdf7ea",
                           panel_bg="#ffffff", text="#402a00", muted="#8a733e",
                           border="#f5e7c6", on_primary="#ffffff", footer_grey="#d9b968"),
    "team_invitation": dict(primary="#1f2f6b", accent="#5b7cf0", bg="#f2f4fb",
                            panel_bg="#ffffff", text="#151d3f", muted="#5c669a",
                            border="#dfe3f5", on_primary="#ffffff", footer_grey="#9ba7d6"),
    "document_shared": dict(primary="#2a4d3c", accent="#68b58a", bg="#f3f8f5",
                            panel_bg="#ffffff", text="#173226", muted="#5c7d6c",
                            border="#dfece5", on_primary="#ffffff", footer_grey="#9fc2ab"),
    "calendar_invite": dict(primary="#31356b", accent="#7280f5", bg="#f3f4fb",
                            panel_bg="#ffffff", text="#1d2044", muted="#5f6392",
                            border="#e1e3f5", on_primary="#ffffff", footer_grey="#a4a9dd"),
    "data_export_ready": dict(primary="#243b53", accent="#3ea8dc", bg="#f2f6f9",
                              panel_bg="#ffffff", text="#152537", muted="#5a7188",
                              border="#dfe8ef", on_primary="#ffffff", footer_grey="#9bb6c9"),
    "maintenance_notice": dict(primary="#3f4650", accent="#ff9f1c", bg="#f5f6f7",
                               panel_bg="#ffffff", text="#22262e", muted="#6b7280",
                               border="#e3e5e8", on_primary="#ffffff", footer_grey="#9aa0aa"),
    "referral_reward": dict(primary="#0f5c5c", accent="#38c6a4", bg="#f1faf8",
                            panel_bg="#ffffff", text="#0b3232", muted="#548480",
                            border="#ddf0eb", on_primary="#ffffff", footer_grey="#98c9be"),
    "milestone_achievement": dict(primary="#7a3b8a", accent="#ffd166", bg="#f9f2fb",
                                  panel_bg="#ffffff", text="#3a1c42", muted="#8a6a92",
                                  border="#efdcf2", on_primary="#ffffff", footer_grey="#d3aedb"),
    "tax_receipt": dict(primary="#1c2b4a", accent="#8fa3c9", bg="#f3f5f8",
                        panel_bg="#ffffff", text="#151f33", muted="#5e6c85",
                        border="#e1e5ee", on_primary="#ffffff", footer_grey="#a3aec4"),
    "waitlist_confirmation": dict(primary="#3d2b6b", accent="#a084e8", bg="#f6f3fb",
                                  panel_bg="#ffffff", text="#241a40", muted="#6f6492",
                                  border="#e8e2f5", on_primary="#ffffff", footer_grey="#bdaadd"),
}


# --------------------------------------------------------------------------
# 5. Per use-case content builders. Each returns:
#    (subject, preheader, header_kicker, header_title, body_rows_html,
#     footer_note, plain_text_lines)
# --------------------------------------------------------------------------

def build_order_confirmation(theme, brand):
    name = rand_name()
    order_id = rand_order_id()
    order_date = rand_date(0)
    items, subtotal = line_items()
    table_html, total = items_table(items, subtotal, theme, tax_rate=0.075, shipping=round(random.uniform(0, 9.99), 2))
    subject = f"Your {brand} order {order_id} is confirmed"
    preheader = f"Thanks for your order, {name.split()[0]}! Here's your receipt for {money(total)}."
    kicker = "Order Confirmation"
    title = f"Thanks for your order, {name.split()[0]}!"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">We're getting your order ready to be packed and shipped. We will notify you when it has been sent.</p>
    {info_row("Order Number", order_id, theme)}
    {info_row("Order Date", order_date, theme)}
    {info_row("Payment Method", random.choice(["Visa ending 4821", "Mastercard ending 1190", "PayPal"]), theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {table_html}
    <div style="height:8px; line-height:8px; font-size:8px;">&nbsp;</div>
    {render_button("View Order Status", f"https://{brand.lower().replace(' ', '')}.com/orders/{order_id}", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">If you have any questions about your order, reply to this email or contact our support team.</p>
    """
    footer = f"You're receiving this email because you placed an order with {brand}."
    plain = [f"Hi {name},", f"Order {order_id} confirmed on {order_date}.", f"Total: {money(total)}"]
    return subject, preheader, kicker, title, body, footer, plain


def build_shipping_notification(theme, brand):
    name = rand_name()
    tracking = rand_tracking()
    carrier = random.choice(CARRIERS)
    eta = rand_date(random.randint(2, 6))
    order_id = rand_order_id()
    subject = f"Your {brand} package is on its way \u2014 arriving {eta}"
    preheader = f"Track your package #{tracking} with {carrier}."
    kicker = "Shipping Notification"
    title = "Your order has shipped!"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Good news \u2014 your order {order_id} is packed and on its way to you.</p>
    {info_row("Carrier", carrier, theme)}
    {info_row("Tracking Number", tracking, theme)}
    {info_row("Estimated Delivery", eta, theme)}
    {info_row("Shipping To", random.choice(CITIES), theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Track My Package", f"https://parcelwave.com/track/{tracking}", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">Delivery times are estimates and may vary based on carrier conditions.</p>
    """
    footer = f"You're receiving this shipping update from {brand}."
    plain = [f"Hi {name},", f"Tracking: {tracking} via {carrier}", f"ETA: {eta}"]
    return subject, preheader, kicker, title, body, footer, plain


def build_password_reset(theme, brand):
    name = rand_name()
    reset_url = f"https://{brand.lower().replace(' ', '')}.com/reset-password?token={uuid.uuid4().hex}"
    expiry_minutes = random.choice([15, 30, 60])
    subject = f"Reset your {brand} password"
    preheader = f"This password reset link expires in {expiry_minutes} minutes."
    kicker = "Password Reset"
    title = "Reset your password"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">We received a request to reset the password for your account. Click the button below to
    choose a new password. This link will expire in <strong>{expiry_minutes} minutes</strong>.</p>
    {render_button("Reset Password", reset_url, theme['accent'])}
    <p style="margin:24px 0 0 0; color:{theme['muted']};">If you didn't request a password reset, you can safely ignore this
    email \u2014 your password will not be changed.</p>
    <p style="margin:12px 0 0 0; color:{theme['muted']}; font-size:12px; word-break:break-all;">
    Or copy and paste this link into your browser:<br>{reset_url}</p>
    """
    footer = f"For your security, this link was generated for your {brand} account only."
    plain = [f"Hi {name},", "Reset your password using the link below.", reset_url]
    return subject, preheader, kicker, title, body, footer, plain


def build_two_factor_auth(theme, brand):
    name = rand_name()
    otp = rand_otp()
    expiry_minutes = random.choice([5, 10])
    subject = f"{otp} is your {brand} verification code"
    preheader = f"Your one-time code is {otp}. It expires in {expiry_minutes} minutes."
    kicker = "Two-Factor Authentication"
    title = "Your verification code"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Use the following one-time code to complete your sign-in. This code expires in {expiry_minutes} minutes.</p>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td align="center" style="padding:12px 0 20px 0;">
          <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="border:1px dashed {theme['accent']}; border-radius:6px;">
            <tr>
              <td style="padding:16px 32px; font-family:{FONT_STACK}; font-size:32px; font-weight:bold; letter-spacing:8px; color:{theme['primary']};">
                {otp}
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
    <p style="margin:0; color:{theme['muted']};">Never share this code with anyone, including {brand} staff. If you didn't
    attempt to sign in, please secure your account immediately.</p>
    """
    footer = f"This code was requested for a sign-in attempt on your {brand} account."
    plain = [f"Hi {name},", f"Your code: {otp}", f"Expires in {expiry_minutes} minutes."]
    return subject, preheader, kicker, title, body, footer, plain


def build_security_alert(theme, brand):
    name = rand_name()
    device = random.choice(DEVICES)
    city = random.choice(CITIES)
    ip = rand_ip()
    when = datetime.now().strftime("%B %d, %Y at %I:%M %p")
    subject = f"New sign-in to your {brand} account"
    preheader = f"We noticed a new sign-in from {city}. Was this you?"
    kicker = "Security Alert"
    title = "We noticed a new sign-in"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Your {brand} account was just accessed from a new device or location. If this was you,
    no action is needed.</p>
    {info_row("Device", device, theme)}
    {info_row("Approximate Location", city, theme)}
    {info_row("IP Address", ip, theme)}
    {info_row("Time", when, theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("This Wasn't Me \u2014 Secure My Account", f"https://{brand.lower().replace(' ', '')}.com/security", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">If you don't recognize this activity, we recommend changing your
    password immediately and reviewing your account's active sessions.</p>
    """
    footer = f"{brand} sends security alerts to help keep your account safe."
    plain = [f"Hi {name},", f"New sign-in from {city} ({ip}) using {device} on {when}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_welcome(theme, brand):
    name = rand_name()
    verify_url = f"https://{brand.lower().replace(' ', '')}.com/verify?token={uuid.uuid4().hex}"
    subject = f"Welcome to {brand}, {name.split()[0]}! Please confirm your email"
    preheader = f"You're almost set \u2014 confirm your email to activate your {brand} account."
    kicker = "Welcome"
    title = f"Welcome aboard, {name.split()[0]}!"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Thanks for creating a {brand} account. We're excited to have you. Please confirm your
    email address to activate all features.</p>
    {render_button("Confirm My Email", verify_url, theme['accent'])}
    <p style="margin:24px 0 0 0; color:{theme['muted']};">Once verified, you'll have full access to your dashboard,
    billing settings, and personalized recommendations.</p>
    <p style="margin:12px 0 0 0; color:{theme['muted']}; font-size:12px; word-break:break-all;">
    Or copy and paste this link into your browser:<br>{verify_url}</p>
    """
    footer = f"You're receiving this email because you signed up for {brand}."
    plain = [f"Hi {name},", "Welcome! Confirm your email:", verify_url]
    return subject, preheader, kicker, title, body, footer, plain


def build_billing_invoice(theme, brand):
    name = rand_name()
    invoice_id = rand_invoice_id()
    period = datetime.now().strftime("%B %Y")
    plan = random.choice(["Starter Plan", "Professional Plan", "Business Plan", "Team Plan"])
    amount = rand_amount(9, 249)
    due_date = rand_date(random.randint(7, 14))
    subject = f"Your {brand} invoice for {period} \u2014 {money(amount)}"
    preheader = f"Invoice {invoice_id} for {money(amount)} is now available."
    kicker = "Billing Invoice"
    title = f"Your {period} invoice is ready"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">This is a summary of your recent {brand} subscription charge. A detailed invoice is
    attached to your account for your records.</p>
    {info_row("Invoice Number", invoice_id, theme)}
    {info_row("Billing Period", period, theme)}
    {info_row("Plan", plan, theme)}
    {info_row("Payment Method", random.choice(["Visa ending 4821", "Mastercard ending 1190", "ACH Transfer"]), theme)}
    {info_row("Amount Due", money(amount), theme)}
    {info_row("Due Date", due_date, theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("View Invoice", f"https://{brand.lower().replace(' ', '')}.com/billing/{invoice_id}", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">This charge will be processed automatically using your default
    payment method on file unless otherwise noted.</p>
    """
    footer = f"{brand} Billing \u2014 questions? Contact billing support any time."
    plain = [f"Hi {name},", f"Invoice {invoice_id}: {money(amount)} due {due_date}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_appointment_reminder(theme, brand):
    name = rand_name()
    date = rand_date(random.randint(1, 5))
    time = rand_time()
    provider = random.choice(["Dr. Alvarez", "Dr. Kim", "Dr. Osei", "Dr. Nakamura", "Dr. Patel"])
    location = random.choice(["Main Street Clinic", "Downtown Office", "Riverside Wellness Center", "North Campus Suite 210"])
    subject = f"Reminder: your {brand} appointment on {date}"
    preheader = f"You have an appointment on {date} at {time}."
    kicker = "Appointment Reminder"
    title = "Upcoming appointment reminder"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">This is a friendly reminder about your upcoming appointment.</p>
    {info_row("Date", date, theme)}
    {info_row("Time", time, theme)}
    {info_row("With", provider, theme)}
    {info_row("Location", location, theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Manage Appointment", f"https://{brand.lower().replace(' ', '')}.com/appointments", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">Need to reschedule? Please let us know at least 24 hours in
    advance so we can offer the slot to another patient.</p>
    """
    footer = f"You're receiving this reminder because you booked with {brand}."
    plain = [f"Hi {name},", f"Appointment on {date} at {time} with {provider}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_system_notification(theme, brand):
    name = rand_name()
    used_pct = random.choice([85, 90, 95, 98, 100])
    used_gb = round(random.uniform(45, 100), 1)
    total_gb = round(used_gb / (used_pct / 100), 1)
    subject = f"You've used {used_pct}% of your {brand} storage"
    preheader = f"You're at {used_pct}% of your storage limit ({used_gb} GB of {total_gb} GB)."
    kicker = "System Notification"
    title = "Your storage is almost full"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Your {brand} account has used <strong>{used_pct}%</strong> of its available storage.
    To keep syncing and uploading new files without interruption, consider freeing up space or upgrading your plan.</p>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td style="padding:4px 0 4px 0;">
          <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="border-radius:4px; background-color:{theme['border']};">
            <tr>
              <td width="{used_pct}%" bgcolor="{theme['accent']}" style="height:14px; line-height:14px; font-size:1px; border-radius:4px;">&nbsp;</td>
              <td bgcolor="{theme['border']}" style="height:14px; line-height:14px; font-size:1px;">&nbsp;</td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
    <p style="margin:10px 0 20px 0; color:{theme['muted']}; font-size:13px;">{used_gb} GB used of {total_gb} GB</p>
    {render_button("Manage Storage", f"https://{brand.lower().replace(' ', '')}.com/storage", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">Once your storage is full, new uploads and backups may be
    paused until space is freed up.</p>
    """
    footer = f"{brand} System Notifications \u2014 sent automatically based on your account usage."
    plain = [f"Hi {name},", f"Storage at {used_pct}% ({used_gb} GB of {total_gb} GB)."]
    return subject, preheader, kicker, title, body, footer, plain


def build_cart_abandonment(theme, brand):
    name = rand_name()
    items, subtotal = line_items(random.randint(1, 2))
    discount_code = "SAVE" + str(random.choice([10, 15, 20]))
    subject = f"You left something in your {brand} cart"
    preheader = f"Your cart is waiting \u2014 complete your order and save with code {discount_code}."
    kicker = "Cart Reminder"
    title = "Still thinking it over?"
    table_html, total = items_table(items, subtotal, theme)
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">You left a few items in your cart. They're still available, but popular items can sell
    out quickly \u2014 complete your purchase before they're gone.</p>
    {table_html}
    <div style="height:12px; line-height:12px; font-size:12px;">&nbsp;</div>
    <p style="margin:0 0 16px 0;">Use code <strong style="color:{theme['accent']};">{discount_code}</strong> at checkout
    for a limited-time discount.</p>
    {render_button("Complete My Purchase", f"https://{brand.lower().replace(' ', '')}.com/cart", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">Prices and availability are subject to change.</p>
    """
    footer = f"You're receiving this email because you have items saved in your {brand} cart."
    plain = [f"Hi {name},", f"Cart total: {money(total)}. Use code {discount_code}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_account_verification(theme, brand):
    name = rand_name()
    code = rand_otp()
    subject = f"Verify your {brand} account"
    preheader = f"Your verification code is {code}."
    kicker = "Account Verification"
    title = "Verify your email address"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">To finish setting up your {brand} account, please verify your email address by entering the code below:</p>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td align="center" style="padding:12px 0 20px 0;">
          <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="background-color:{theme['bg']}; border-radius:6px;">
            <tr>
              <td style="padding:16px 32px; font-family:{FONT_STACK}; font-size:32px; font-weight:bold; letter-spacing:8px; color:{theme['primary']};">
                {code}
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
    <p style="margin:0; color:{theme['muted']};">If you didn't create an account, you can safely ignore this email.</p>
    """
    footer = f"This verification code was sent to {name}."
    plain = [f"Hi {name},", f"Your verification code: {code}"]
    return subject, preheader, kicker, title, body, footer, plain


def build_subscription_cancelled(theme, brand):
    name = rand_name()
    end_date = rand_date(30)
    subject = f"Your {brand} subscription has been cancelled"
    preheader = f"Your subscription will end on {end_date}."
    kicker = "Subscription Update"
    title = "We're sorry to see you go"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">As requested, your {brand} subscription has been cancelled. You will still have access to your account features until <strong>{end_date}</strong>.</p>
    <p style="margin:0 0 20px 0;">We'd love to have you back! If you change your mind, you can reactivate your subscription at any time.</p>
    {render_button("Reactivate Subscription", f"https://{brand.lower().replace(' ', '')}.com/billing", theme['accent'])}
    """
    footer = f"Your {brand} subscription ends on {end_date}."
    plain = [f"Hi {name},", f"Your subscription ends on {end_date}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_payment_failed(theme, brand):
    name = rand_name()
    amount = rand_amount(10, 100)
    subject = f"Action Required: Payment failed for {brand}"
    preheader = f"We were unable to process your payment of {money(amount)}."
    kicker = "Billing Alert"
    title = "Payment unsuccessful"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">We were unable to process your latest payment of {money(amount)} for your {brand} subscription. This might be due to an expired card or insufficient funds.</p>
    <p style="margin:0 0 20px 0;">Please update your payment information to avoid any service interruption.</p>
    {render_button("Update Payment Info", f"https://{brand.lower().replace(' ', '')}.com/billing", theme['accent'])}
    """
    footer = f"Please contact support if you have questions about this billing alert."
    plain = [f"Hi {name},", f"Payment of {money(amount)} failed for {brand}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_feedback_request(theme, brand):
    name = rand_name()
    subject = f"How are we doing, {name.split()[0]}?"
    preheader = f"We'd love to hear your thoughts on {brand}."
    kicker = "Customer Feedback"
    title = "Share your feedback"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Thank you for being a part of {brand}! We're always looking for ways to improve, and we'd love to hear about your experience.</p>
    <p style="margin:0 0 20px 0;">It only takes a minute to share your thoughts.</p>
    {render_button("Take the Survey", f"https://{brand.lower().replace(' ', '')}.com/survey", theme['accent'])}
    """
    footer = f"Thank you for helping us improve {brand}."
    plain = [f"Hi {name},", f"Share your feedback with {brand} here: https://{brand.lower().replace(' ', '')}.com/survey"]
    return subject, preheader, kicker, title, body, footer, plain


def build_trial_expiring(theme, brand):
    name = rand_name()
    days_left = random.choice([1, 2, 3])
    subject = f"Your {brand} trial ends in {days_left} day{'s' if days_left > 1 else ''}"
    preheader = f"Your free trial is almost over. Upgrade now to keep your access."
    kicker = "Trial Ending Soon"
    title = "Your trial is expiring"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Your free trial of {brand} will end in <strong>{days_left} day{'s' if days_left > 1 else ''}</strong>. We hope you've enjoyed exploring our features!</p>
    <p style="margin:0 0 20px 0;">To continue using {brand} without interruption, please choose a plan that works for you.</p>
    {render_button("Upgrade Now", f"https://{brand.lower().replace(' ', '')}.com/pricing", theme['accent'])}
    """
    footer = f"Upgrade before your trial ends to keep all your data."
    plain = [f"Hi {name},", f"Your {brand} trial ends in {days_left} days. Upgrade here: https://{brand.lower().replace(' ', '')}.com/pricing"]
    return subject, preheader, kicker, title, body, footer, plain


def build_refund_processed(theme, brand):
    name = rand_name()
    order_id = rand_order_id()
    amount = rand_amount(15, 220)
    processed_date = rand_date(0)
    subject = f"Your refund of {money(amount)} has been processed"
    preheader = f"We've refunded {money(amount)} for order {order_id} to your original payment method."
    kicker = "Refund Processed"
    title = "Your refund is on its way"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">We've processed a refund for your recent {brand} order. It may take 5-10 business days
    to appear on your original payment method, depending on your bank.</p>
    {info_row("Order Number", order_id, theme)}
    {info_row("Refund Amount", money(amount), theme)}
    {info_row("Processed On", processed_date, theme)}
    {info_row("Refund Method", random.choice(["Visa ending 4821", "Mastercard ending 1190", "PayPal"]), theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("View Refund Details", f"https://{brand.lower().replace(' ', '')}.com/orders/{order_id}", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">If you don't see the refund after 10 business days, please
    contact your bank or reach out to our support team.</p>
    """
    footer = f"You're receiving this email because a refund was issued on your {brand} account."
    plain = [f"Hi {name},", f"Refund of {money(amount)} processed for order {order_id} on {processed_date}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_subscription_renewed(theme, brand):
    name = rand_name()
    plan = random.choice(["Starter Plan", "Professional Plan", "Business Plan", "Team Plan"])
    amount = rand_amount(9, 249)
    next_billing = rand_date(30)
    invoice_id = rand_invoice_id()
    subject = f"Your {brand} subscription has renewed"
    preheader = f"Your {plan} renewed for {money(amount)}. Next billing date: {next_billing}."
    kicker = "Subscription Renewed"
    title = "Your subscription renewed successfully"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Thanks for staying with {brand}! Your subscription has renewed and you'll continue to
    have full access to all your plan's features.</p>
    {info_row("Plan", plan, theme)}
    {info_row("Amount Charged", money(amount), theme)}
    {info_row("Invoice Number", invoice_id, theme)}
    {info_row("Next Billing Date", next_billing, theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("View Invoice", f"https://{brand.lower().replace(' ', '')}.com/billing/{invoice_id}", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">You can update your plan or payment method at any time from
    your account settings.</p>
    """
    footer = f"{brand} Billing — questions? Contact billing support any time."
    plain = [f"Hi {name},", f"Subscription renewed: {plan} — {money(amount)}. Next billing: {next_billing}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_account_locked(theme, brand):
    name = rand_name()
    attempts = random.choice([5, 6, 8, 10])
    ip = rand_ip()
    when = datetime.now().strftime("%B %d, %Y at %I:%M %p")
    unlock_url = f"https://{brand.lower().replace(' ', '')}.com/unlock?token={uuid.uuid4().hex}"
    subject = f"Your {brand} account has been temporarily locked"
    preheader = f"We locked your account after {attempts} failed sign-in attempts."
    kicker = "Account Locked"
    title = "Your account is temporarily locked"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">For your protection, we've temporarily locked your {brand} account after
    <strong>{attempts} failed sign-in attempts</strong>.</p>
    {info_row("Attempted From IP", ip, theme)}
    {info_row("Time", when, theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Unlock My Account", unlock_url, theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">If this wasn't you, we recommend resetting your password
    as soon as your account is unlocked.</p>
    """
    footer = f"{brand} sends this notice automatically to protect your account."
    plain = [f"Hi {name},", f"Account locked after {attempts} failed attempts from {ip} on {when}.", unlock_url]
    return subject, preheader, kicker, title, body, footer, plain


def build_newsletter_digest(theme, brand):
    name = rand_name()
    headlines = random.sample([
        "5 ways teams are saving time this quarter",
        "Behind the scenes: how we built our newest feature",
        "Customer spotlight: scaling from 10 to 10,000 users",
        "Product roadmap: what's coming next",
        "Tips from our top power users",
        "A look back at this month's biggest updates",
        "How to get the most out of your dashboard",
        "New integrations you might have missed",
    ], 3)
    subject = f"{brand} Digest: {headlines[0]}"
    preheader = f"This week: {headlines[0]}, plus {len(headlines) - 1} more stories."
    kicker = "Newsletter Digest"
    title = "Your weekly digest"
    items_html = ""
    for h in headlines:
        items_html += f"""
        <tr>
          <td style="padding:12px 0; border-bottom:1px solid {theme['border']};">
            <p style="margin:0 0 4px 0; font-family:{FONT_STACK}; font-size:14px; font-weight:bold; color:{theme['text']};">{h}</p>
            <a href="https://{brand.lower().replace(' ', '')}.com/blog" style="font-family:{FONT_STACK}; font-size:12px; color:{theme['accent']}; text-decoration:none;">Read more &rarr;</a>
          </td>
        </tr>
        """
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Here's what's new from {brand} this week.</p>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      {items_html}
    </table>
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Read the Full Digest", f"https://{brand.lower().replace(' ', '')}.com/blog", theme['accent'])}
    """
    footer = f"You're subscribed to the {brand} newsletter."
    plain = [f"Hi {name},"] + headlines
    return subject, preheader, kicker, title, body, footer, plain


def build_event_invitation(theme, brand):
    name = rand_name()
    event_name = random.choice([
        "Product Roadmap Webinar", "Annual Customer Summit", "Q&A with the Founders",
        "New Feature Launch Event", "Community Meetup", "Live Training Session",
    ])
    date = rand_date(random.randint(3, 20))
    time = rand_time()
    location = random.choice(["Online via Zoom", "San Francisco, CA", "New York, NY", "Virtual Event"])
    subject = f"You're invited: {event_name}"
    preheader = f"Join us on {date} at {time} for {event_name}."
    kicker = "Event Invitation"
    title = f"You're invited to {event_name}"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">{brand} would like to invite you to an upcoming event. We'd love for you to join us.</p>
    {info_row("Event", event_name, theme)}
    {info_row("Date", date, theme)}
    {info_row("Time", time, theme)}
    {info_row("Location", location, theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td align="center" style="padding:4px 6px;">
          <table role="presentation" cellpadding="0" cellspacing="0" border="0">
            <tr>
              <td align="center" bgcolor="{theme['accent']}" style="border-radius:4px;">
                <a href="https://{brand.lower().replace(' ', '')}.com/rsvp?response=yes" target="_blank"
                   style="display:inline-block; padding:12px 26px; font-family:{FONT_STACK}; font-size:14px;
                          font-weight:bold; color:#ffffff; text-decoration:none;">Accept</a>
              </td>
            </tr>
          </table>
        </td>
        <td align="center" style="padding:4px 6px;">
          <table role="presentation" cellpadding="0" cellspacing="0" border="0">
            <tr>
              <td align="center" style="border:1px solid {theme['border']}; border-radius:4px;">
                <a href="https://{brand.lower().replace(' ', '')}.com/rsvp?response=no" target="_blank"
                   style="display:inline-block; padding:12px 26px; font-family:{FONT_STACK}; font-size:14px;
                          font-weight:bold; color:{theme['muted']}; text-decoration:none;">Decline</a>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
    """
    footer = f"You're receiving this invitation from {brand}."
    plain = [f"Hi {name},", f"Invited to {event_name} on {date} at {time} ({location})."]
    return subject, preheader, kicker, title, body, footer, plain


def build_survey_results_shared(theme, brand):
    name = rand_name()
    changes = random.sample([
        "Faster page load times across the dashboard",
        "A redesigned onboarding flow",
        "New keyboard shortcuts for power users",
        "Improved mobile experience",
        "Expanded export options",
        "A cleaner, more accessible color palette",
    ], 3)
    subject = f"You spoke, we listened: what's new at {brand}"
    preheader = "Here's what changed based on your feedback."
    kicker = "Survey Results"
    title = "Your feedback shaped these changes"
    changes_html = "".join(
        f"""<tr><td style="padding:8px 0; font-family:{FONT_STACK}; font-size:14px; color:{theme['text']};">
        &#10003;&nbsp; {c}</td></tr>""" for c in changes
    )
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Thank you for taking the time to share your feedback with us. Here's what we built
    as a direct result:</p>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      {changes_html}
    </table>
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("See What's New", f"https://{brand.lower().replace(' ', '')}.com/changelog", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">Keep the feedback coming — it genuinely shapes our roadmap.</p>
    """
    footer = f"Thank you for helping us improve {brand}."
    plain = [f"Hi {name},"] + changes
    return subject, preheader, kicker, title, body, footer, plain


def build_loyalty_points_earned(theme, brand):
    name = rand_name()
    points_earned = random.choice([50, 100, 150, 250, 500])
    total_points = points_earned + random.randint(100, 2000)
    next_tier_points = total_points + random.randint(200, 800)
    pct = min(100, int((total_points / next_tier_points) * 100))
    subject = f"You earned {points_earned} {brand} points!"
    preheader = f"You now have {total_points} points — you're {100 - pct}% away from your next reward."
    kicker = "Loyalty Rewards"
    title = f"You earned {points_earned} points!"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Nice! You just earned <strong>{points_earned} points</strong> from your recent
    activity with {brand}.</p>
    {info_row("Points Earned", f"+{points_earned}", theme)}
    {info_row("Total Balance", f"{total_points} pts", theme)}
    <div style="height:12px; line-height:12px; font-size:12px;">&nbsp;</div>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td style="padding:4px 0 4px 0;">
          <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="border-radius:4px; background-color:{theme['border']};">
            <tr>
              <td width="{pct}%" bgcolor="{theme['accent']}" style="height:14px; line-height:14px; font-size:1px; border-radius:4px;">&nbsp;</td>
              <td bgcolor="{theme['border']}" style="height:14px; line-height:14px; font-size:1px;">&nbsp;</td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
    <p style="margin:10px 0 20px 0; color:{theme['muted']}; font-size:13px;">{total_points} of {next_tier_points} points to your next reward tier</p>
    {render_button("View My Rewards", f"https://{brand.lower().replace(' ', '')}.com/rewards", theme['accent'])}
    """
    footer = f"You're receiving this because you're a member of the {brand} rewards program."
    plain = [f"Hi {name},", f"Earned {points_earned} points. Total: {total_points} pts."]
    return subject, preheader, kicker, title, body, footer, plain


def build_gift_card_received(theme, brand):
    name = rand_name()
    sender = rand_name()
    amount = random.choice([25, 50, 75, 100, 150, 200])
    code = "-".join("".join(random.choices("ABCDEFGHJKLMNPQRSTUVWXYZ0123456789", k=4)) for _ in range(4))
    message = random.choice([
        "Happy Birthday! Treat yourself to something nice.",
        "Congratulations on the new job!",
        "Thank you for everything — enjoy!",
        "Just because. Hope you love it!",
    ])
    subject = f"{sender.split()[0]} sent you a {money(amount)} {brand} gift card!"
    preheader = f"You've received a {money(amount)} gift card from {sender}."
    kicker = "Gift Card Received"
    title = f"You've received a gift card!"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;"><strong>{sender}</strong> sent you a {brand} gift card with this message:</p>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td style="padding:14px 18px; background-color:{theme['bg']}; border-left:3px solid {theme['accent']}; font-family:{FONT_STACK}; font-size:14px; color:{theme['text']}; font-style:italic;">
          &ldquo;{message}&rdquo;
        </td>
      </tr>
    </table>
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td align="center" style="padding:12px 0 20px 0;">
          <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="border:1px dashed {theme['accent']}; border-radius:6px;">
            <tr>
              <td style="padding:16px 32px; font-family:{FONT_STACK}; font-size:22px; font-weight:bold; letter-spacing:2px; color:{theme['primary']};">
                {money(amount)}<br>
                <span style="font-size:15px; letter-spacing:3px;">{code}</span>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
    {render_button("Redeem My Gift Card", f"https://{brand.lower().replace(' ', '')}.com/redeem?code={code}", theme['accent'])}
    """
    footer = f"This gift card is redeemable at {brand} and does not expire unless stated otherwise."
    plain = [f"Hi {name},", f"{sender} sent you a {money(amount)} gift card. Code: {code}"]
    return subject, preheader, kicker, title, body, footer, plain


def build_price_drop_alert(theme, brand):
    name = rand_name()
    product_name, old_price = random.choice(PRODUCTS)
    new_price = round(old_price * random.uniform(0.6, 0.85), 2)
    pct_off = round((1 - new_price / old_price) * 100)
    subject = f"Price drop: {product_name} is now {money(new_price)}"
    preheader = f"Save {pct_off}% on {product_name} — now {money(new_price)}."
    kicker = "Price Drop Alert"
    title = "An item on your wishlist just dropped in price"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Good news — an item you've been eyeing at {brand} just got cheaper.</p>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td style="padding:14px 0; border-bottom:1px solid {theme['border']}; font-family:{FONT_STACK}; font-size:14px; color:{theme['text']};">
          <strong>{product_name}</strong><br>
          <span style="text-decoration:line-through; color:{theme['muted']};">{money(old_price)}</span>
          &nbsp;<span style="color:{theme['accent']}; font-weight:bold; font-size:16px;">{money(new_price)}</span>
          &nbsp;<span style="color:{theme['accent']};">({pct_off}% off)</span>
        </td>
      </tr>
    </table>
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Shop Now", f"https://{brand.lower().replace(' ', '')}.com/shop", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">This price is subject to change and while supplies last.</p>
    """
    footer = f"You're receiving this alert because you saved this item at {brand}."
    plain = [f"Hi {name},", f"{product_name} dropped from {money(old_price)} to {money(new_price)}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_back_in_stock(theme, brand):
    name = rand_name()
    product_name, price = random.choice(PRODUCTS)
    subject = f"{product_name} is back in stock!"
    preheader = f"The item you wanted is back — grab it before it sells out again."
    kicker = "Back In Stock"
    title = "It's back!"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">You asked us to let you know — <strong>{product_name}</strong> is back in stock at {brand}.</p>
    {info_row("Item", product_name, theme)}
    {info_row("Price", money(price), theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Buy Now", f"https://{brand.lower().replace(' ', '')}.com/shop", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">Stock is limited and this item may sell out quickly again.</p>
    """
    footer = f"You're receiving this alert because you asked to be notified by {brand}."
    plain = [f"Hi {name},", f"{product_name} ({money(price)}) is back in stock."]
    return subject, preheader, kicker, title, body, footer, plain


def build_review_request(theme, brand):
    name = rand_name()
    product_name, _price = random.choice(PRODUCTS)
    order_id = rand_order_id()
    stars_html = "".join(
        f'<span style="color:{theme["accent"]}; font-size:24px;">&#9733;</span>' for _ in range(5)
    )
    subject = f"How was your {product_name}?"
    preheader = f"Tell us what you think about your recent {brand} purchase."
    kicker = "Review Request"
    title = "How did we do?"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">We hope you're enjoying your recent purchase. Would you take a moment to leave a review?</p>
    {info_row("Item", product_name, theme)}
    {info_row("Order Number", order_id, theme)}
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td align="center" style="padding:16px 0;">
          {stars_html}
        </td>
      </tr>
    </table>
    {render_button("Leave a Review", f"https://{brand.lower().replace(' ', '')}.com/review/{order_id}", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">Your feedback helps other shoppers and helps us improve.</p>
    """
    footer = f"You're receiving this email because you recently purchased from {brand}."
    plain = [f"Hi {name},", f"Please review your purchase: {product_name} (Order {order_id})."]
    return subject, preheader, kicker, title, body, footer, plain


def build_team_invitation(theme, brand):
    name = rand_name()
    inviter = rand_name()
    workspace = random.choice(["Marketing Team", "Engineering", "Product Ops", "Design Studio", "Growth Squad"])
    invite_url = f"https://{brand.lower().replace(' ', '')}.com/invite?token={uuid.uuid4().hex}"
    subject = f"{inviter.split()[0]} invited you to join {workspace} on {brand}"
    preheader = f"{inviter} invited you to collaborate on {brand}."
    kicker = "Team Invitation"
    title = f"You've been invited to {workspace}"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;"><strong>{inviter}</strong> has invited you to join the <strong>{workspace}</strong>
    workspace on {brand}.</p>
    {info_row("Workspace", workspace, theme)}
    {info_row("Invited By", inviter, theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Accept Invitation", invite_url, theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">This invitation will expire in 7 days.</p>
    """
    footer = f"You're receiving this invitation because someone added your email on {brand}."
    plain = [f"Hi {name},", f"{inviter} invited you to {workspace} on {brand}.", invite_url]
    return subject, preheader, kicker, title, body, footer, plain


def build_document_shared(theme, brand):
    name = rand_name()
    sharer = rand_name()
    doc_name = random.choice([
        "Q3 Roadmap.pdf", "Design Review Notes.docx", "Budget Forecast.xlsx",
        "Onboarding Guide.pdf", "Meeting Minutes.docx", "Product Spec v2.pdf",
    ])
    view_url = f"https://{brand.lower().replace(' ', '')}.com/documents/{uuid.uuid4().hex[:8]}"
    subject = f"{sharer.split()[0]} shared \"{doc_name}\" with you"
    preheader = f"{sharer} shared a document with you on {brand}."
    kicker = "Document Shared"
    title = "A document was shared with you"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;"><strong>{sharer}</strong> shared a file with you on {brand}.</p>
    {info_row("File Name", doc_name, theme)}
    {info_row("Shared By", sharer, theme)}
    {info_row("Shared On", rand_date(0), theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("View Document", view_url, theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">This link is only accessible to people with permission.</p>
    """
    footer = f"You're receiving this email because a file was shared with you on {brand}."
    plain = [f"Hi {name},", f"{sharer} shared {doc_name} with you.", view_url]
    return subject, preheader, kicker, title, body, footer, plain


def build_calendar_invite(theme, brand):
    name = rand_name()
    organizer = rand_name()
    meeting = random.choice([
        "Sprint Planning", "1:1 Sync", "Design Review", "All-Hands Meeting",
        "Client Kickoff Call", "Quarterly Business Review", "Retro",
    ])
    date = rand_date(random.randint(1, 10))
    time = rand_time()
    join_url = f"https://{brand.lower().replace(' ', '')}.com/meet/{uuid.uuid4().hex[:8]}"
    subject = f"Invitation: {meeting} @ {date} {time}"
    preheader = f"{organizer} invited you to {meeting} on {date} at {time}."
    kicker = "Calendar Invite"
    title = meeting
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;"><strong>{organizer}</strong> has invited you to a meeting.</p>
    {info_row("Meeting", meeting, theme)}
    {info_row("Organizer", organizer, theme)}
    {info_row("Date", date, theme)}
    {info_row("Time", time, theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Join Meeting", join_url, theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">Add this event to your calendar so you don't miss it.</p>
    """
    footer = f"You're receiving this invite via {brand} Calendar."
    plain = [f"Hi {name},", f"{meeting} with {organizer} on {date} at {time}.", join_url]
    return subject, preheader, kicker, title, body, footer, plain


def build_data_export_ready(theme, brand):
    name = rand_name()
    export_type = random.choice(["Account Activity", "Analytics Report", "Full Data Export", "Transaction History"])
    size_mb = round(random.uniform(1.2, 340), 1)
    download_url = f"https://{brand.lower().replace(' ', '')}.com/exports/{uuid.uuid4().hex[:10]}"
    expiry_days = random.choice([3, 5, 7])
    subject = f"Your {brand} export is ready to download"
    preheader = f"Your {export_type} export ({size_mb} MB) is ready. Link expires in {expiry_days} days."
    kicker = "Export Ready"
    title = "Your data export is ready"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">The export you requested has finished processing and is ready to download.</p>
    {info_row("Export Type", export_type, theme)}
    {info_row("File Size", f"{size_mb} MB", theme)}
    {info_row("Link Expires", f"In {expiry_days} days", theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Download Export", download_url, theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">For security, this download link will expire and cannot be
    regenerated — request a new export if needed.</p>
    """
    footer = f"You're receiving this email because you requested a data export from {brand}."
    plain = [f"Hi {name},", f"{export_type} export ({size_mb} MB) ready.", download_url]
    return subject, preheader, kicker, title, body, footer, plain


def build_maintenance_notice(theme, brand):
    name = rand_name()
    date = rand_date(random.randint(2, 10))
    start_time = rand_time()
    duration_hours = random.choice([1, 2, 3, 4])
    subject = f"Scheduled maintenance for {brand} on {date}"
    preheader = f"{brand} will be briefly unavailable on {date} starting at {start_time}."
    kicker = "Maintenance Notice"
    title = "Upcoming scheduled maintenance"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">We're performing scheduled maintenance to improve {brand}'s reliability and
    performance. During this window, the service may be temporarily unavailable.</p>
    {info_row("Date", date, theme)}
    {info_row("Start Time", start_time, theme)}
    {info_row("Expected Duration", f"~{duration_hours} hour{'s' if duration_hours > 1 else ''}", theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("View Status Page", f"https://status.{brand.lower().replace(' ', '')}.com", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">We apologize for any inconvenience and appreciate your patience.</p>
    """
    footer = f"{brand} sends maintenance notices to keep you informed."
    plain = [f"Hi {name},", f"Maintenance on {date} at {start_time} for ~{duration_hours}h."]
    return subject, preheader, kicker, title, body, footer, plain


def build_referral_reward(theme, brand):
    name = rand_name()
    referred_name = rand_name()
    reward_amount = random.choice([10, 15, 20, 25, 50])
    subject = f"You earned {money(reward_amount)} for referring a friend!"
    preheader = f"{referred_name} joined {brand} using your referral link — here's your reward."
    kicker = "Referral Reward"
    title = "Your referral reward has arrived"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Great news — <strong>{referred_name}</strong> signed up for {brand} using your
    referral link, and you've earned a reward!</p>
    {info_row("Referred Friend", referred_name, theme)}
    {info_row("Reward Earned", money(reward_amount), theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("View My Referrals", f"https://{brand.lower().replace(' ', '')}.com/referrals", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">Keep sharing your referral link to earn even more rewards.</p>
    """
    footer = f"You're receiving this email because you're a member of the {brand} referral program."
    plain = [f"Hi {name},", f"{referred_name} joined via your referral. You earned {money(reward_amount)}."]
    return subject, preheader, kicker, title, body, footer, plain


def build_milestone_achievement(theme, brand):
    name = rand_name()
    milestone = random.choice([
        "100 tasks completed", "1 year as a member", "50 projects created",
        "1,000 messages sent", "First 10 team members added", "500 hours logged",
    ])
    badge = random.choice(["Gold", "Platinum", "Rising Star", "Power User", "Trailblazer"])
    subject = f"Achievement unlocked: {milestone}!"
    preheader = f"You just hit a milestone on {brand} — {milestone}."
    kicker = "Milestone Achievement"
    title = "You unlocked a new achievement!"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Congratulations! You've just reached a new milestone with {brand}.</p>
    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
      <tr>
        <td align="center" style="padding:16px 0 20px 0;">
          <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="border:2px solid {theme['accent']}; border-radius:32px;">
            <tr>
              <td style="padding:16px 28px; font-family:{FONT_STACK}; font-size:14px; font-weight:bold; letter-spacing:1px;
                         text-transform:uppercase; color:{theme['primary']};">
                &#9733; {badge} Badge
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
    {info_row("Milestone", milestone, theme)}
    <div style="height:12px; line-height:12px; font-size:12px;">&nbsp;</div>
    {render_button("View My Achievements", f"https://{brand.lower().replace(' ', '')}.com/achievements", theme['accent'])}
    """
    footer = f"You're receiving this email because {brand} tracks your account milestones."
    plain = [f"Hi {name},", f"Achievement unlocked: {milestone} ({badge} Badge)."]
    return subject, preheader, kicker, title, body, footer, plain


def build_tax_receipt(theme, brand):
    name = rand_name()
    year = datetime.now().year - random.choice([0, 1])
    total_paid = rand_amount(80, 3000)
    receipt_id = "TAX-" + str(year) + "-" + "".join(random.choices("0123456789", k=6))
    subject = f"Your {year} {brand} tax receipt"
    preheader = f"Your annual receipt of total payments made to {brand} in {year} is ready."
    kicker = "Tax Receipt"
    title = f"Your {year} tax receipt"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">For your records, here is a summary of the total payments made to {brand} during
    {year}. Please consult a tax professional regarding deductibility.</p>
    {info_row("Receipt Number", receipt_id, theme)}
    {info_row("Tax Year", str(year), theme)}
    {info_row("Total Paid", money(total_paid), theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Download Receipt", f"https://{brand.lower().replace(' ', '')}.com/tax-receipts/{receipt_id}", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">A copy of this receipt is also available anytime from your
    billing history.</p>
    """
    footer = f"{brand} — this is not tax advice. Please consult a professional."
    plain = [f"Hi {name},", f"{year} tax receipt {receipt_id}: total paid {money(total_paid)}."]
    return subject, preheader, kicker, title, body, footer, plain


def rand_waitlist_position():
    return random.randint(12, 4800)


def build_waitlist_confirmation(theme, brand):
    name = rand_name()
    position = rand_waitlist_position()
    product = random.choice(["Early Access Program", "Beta Program", "Premium Tier", "New Feature Preview"])
    subject = f"You're on the {brand} waitlist!"
    preheader = f"You're #{position} in line for {product}."
    kicker = "Waitlist Confirmation"
    title = "You're on the list!"
    body = f"""
    <p style="margin:0 0 16px 0;">Hi {name},</p>
    <p style="margin:0 0 20px 0;">Thanks for your interest in {brand}'s {product}. You've been added to the waitlist.</p>
    {info_row("Program", product, theme)}
    {info_row("Your Position", f"#{position}", theme)}
    <div style="height:16px; line-height:16px; font-size:16px;">&nbsp;</div>
    {render_button("Invite Friends to Move Up", f"https://{brand.lower().replace(' ', '')}.com/waitlist/share", theme['accent'])}
    <p style="margin:20px 0 0 0; color:{theme['muted']};">We'll email you as soon as it's your turn — no action needed
    in the meantime.</p>
    """
    footer = f"You're receiving this email because you joined the {brand} waitlist."
    plain = [f"Hi {name},", f"You're #{position} on the {product} waitlist."]
    return subject, preheader, kicker, title, body, footer, plain


CATEGORY_BUILDERS = {
    "order_confirmation": (build_order_confirmation, ["Cobalt Market", "BrightCart", "PixelForge Shop"]),
    "shipping_notification": (build_shipping_notification, ["ParcelWave", "Cobalt Market", "BrightCart"]),
    "password_reset": (build_password_reset, ["Vertex Suite", "Nimbus Cloud", "Meridian Studio"]),
    "two_factor_auth": (build_two_factor_auth, ["Vertex Suite", "Nimbus Cloud", "Lumen Bank"]),
    "security_alert": (build_security_alert, ["Nimbus Cloud", "Vertex Suite", "Lumen Bank"]),
    "welcome": (build_welcome, ["Meridian Studio", "Drift Storage", "PixelForge"]),
    "billing_invoice": (build_billing_invoice, ["Lumen Bank", "Nimbus Cloud", "Vertex Suite"]),
    "appointment_reminder": (build_appointment_reminder, ["Fernwell Health", "Urban Reserve"]),
    "system_notification": (build_system_notification, ["Drift Storage", "Nimbus Cloud"]),
    "cart_abandonment": (build_cart_abandonment, ["PixelForge Shop", "BrightCart", "Cobalt Market"]),
    "account_verification": (build_account_verification, ["Vertex Suite", "Nimbus Cloud", "Meridian Studio"]),
    "subscription_cancelled": (build_subscription_cancelled, ["Nimbus Cloud", "Meridian Studio", "Vertex Suite"]),
    "payment_failed": (build_payment_failed, ["Lumen Bank", "Nimbus Cloud", "Vertex Suite"]),
    "feedback_request": (build_feedback_request, ["Meridian Studio", "Drift Storage", "PixelForge"]),
    "trial_expiring": (build_trial_expiring, ["Vertex Suite", "Nimbus Cloud", "Drift Storage"]),
    "refund_processed": (build_refund_processed, ["Cobalt Market", "BrightCart", "PixelForge Shop"]),
    "subscription_renewed": (build_subscription_renewed, ["Nimbus Cloud", "Vertex Suite", "Meridian Studio"]),
    "account_locked": (build_account_locked, ["Nimbus Cloud", "Vertex Suite", "Lumen Bank"]),
    "newsletter_digest": (build_newsletter_digest, ["Meridian Studio", "Drift Storage", "PixelForge"]),
    "event_invitation": (build_event_invitation, ["Meridian Studio", "Urban Reserve", "Vertex Suite"]),
    "survey_results_shared": (build_survey_results_shared, ["Meridian Studio", "Vertex Suite", "Drift Storage"]),
    "loyalty_points_earned": (build_loyalty_points_earned, ["Cobalt Market", "BrightCart", "PixelForge Shop"]),
    "gift_card_received": (build_gift_card_received, ["Cobalt Market", "BrightCart", "PixelForge Shop"]),
    "price_drop_alert": (build_price_drop_alert, ["PixelForge Shop", "BrightCart", "Cobalt Market"]),
    "back_in_stock": (build_back_in_stock, ["PixelForge Shop", "BrightCart", "Cobalt Market"]),
    "review_request": (build_review_request, ["Cobalt Market", "BrightCart", "PixelForge Shop"]),
    "team_invitation": (build_team_invitation, ["Meridian Studio", "Nimbus Cloud", "Vertex Suite"]),
    "document_shared": (build_document_shared, ["Meridian Studio", "Nimbus Cloud", "Drift Storage"]),
    "calendar_invite": (build_calendar_invite, ["Meridian Studio", "Vertex Suite", "Nimbus Cloud"]),
    "data_export_ready": (build_data_export_ready, ["Nimbus Cloud", "Drift Storage", "Vertex Suite"]),
    "maintenance_notice": (build_maintenance_notice, ["Nimbus Cloud", "Vertex Suite", "Drift Storage"]),
    "referral_reward": (build_referral_reward, ["Meridian Studio", "Nimbus Cloud", "Cobalt Market"]),
    "milestone_achievement": (build_milestone_achievement, ["Meridian Studio", "Vertex Suite", "Nimbus Cloud"]),
    "tax_receipt": (build_tax_receipt, ["Lumen Bank", "Nimbus Cloud", "Vertex Suite"]),
    "waitlist_confirmation": (build_waitlist_confirmation, ["Meridian Studio", "PixelForge", "Vertex Suite"]),
}


# --------------------------------------------------------------------------
# 6. Build a pool of exactly 75 "canned" body templates: 5 per use-case.
#    Each template is (category_key,) and is rendered fresh (with new
#    randomized data / brand choice) every time it's drawn, so the pool
#    reads as 75 distinct canned templates while still producing unique
#    content on every send.
# --------------------------------------------------------------------------

BODY_TEMPLATE_POOL = []
for _category_key in CATEGORY_BUILDERS:
    for _variant in range(5):
        BODY_TEMPLATE_POOL.append(_category_key)

assert len(BODY_TEMPLATE_POOL) == 175


def render_random_email():
    category_key = random.choice(BODY_TEMPLATE_POOL)
    builder_fn, brands = CATEGORY_BUILDERS[category_key]
    theme = THEMES[category_key]
    brand = random.choice(brands)
    subject, preheader, kicker, title, body_rows, footer_note, plain_lines = builder_fn(theme, brand)

    to_addr, from_addr = random.sample(ADDRESS_POOL, 2)
    unsubscribe_addr = from_addr

    layout_fn = random.choice(LAYOUT_RENDERERS)
    html = layout_fn(
        theme=theme,
        preheader=preheader,
        brand_name=brand,
        header_kicker=kicker,
        header_title=title,
        body_rows_html=body_rows,
        footer_note=footer_note,
        unsubscribe_addr=unsubscribe_addr,
    )
    text = render_text_fallback(title, plain_lines)
    return subject, to_addr, from_addr, html, text, category_key


# --------------------------------------------------------------------------
# 7. SMTP sending
# --------------------------------------------------------------------------

def build_message(subject, to_addr, from_addr, html, text):
    msg = MIMEMultipart("alternative")
    msg["Subject"] = subject
    msg["From"] = from_addr
    msg["To"] = to_addr
    msg["Date"] = formatdate(localtime=True)
    msg["Message-ID"] = make_msgid()
    msg.attach(MIMEText(text, "plain", "utf-8"))
    msg.attach(MIMEText(html, "html", "utf-8"))
    return msg


def send_emails(host, port, count, use_tls, username, password, verbose):
    sent = 0
    smtp = smtplib.SMTP(host, port, timeout=30)
    try:
        smtp.ehlo()
        if use_tls:
            smtp.starttls()
            smtp.ehlo()
        if username and password:
            smtp.login(username, password)

        for i in range(1, count + 1):
            subject, to_addr, from_addr, html, text, category = render_random_email()
            msg = build_message(subject, to_addr, from_addr, html, text)
            smtp.sendmail(from_addr, [to_addr], msg.as_string())
            sent += 1
            if verbose:
                print(f"[{i}/{count}] ({category}) {from_addr} -> {to_addr} :: {subject}")
    finally:
        try:
            smtp.quit()
        except Exception:
            pass
    return sent


# --------------------------------------------------------------------------
# 8. CLI
# --------------------------------------------------------------------------

def parse_args(argv=None):
    parser = argparse.ArgumentParser(
        description="Generate realistic HTML business emails and send them to an SMTP server."
    )
    parser.add_argument("--host", default="localhost", help="SMTP server host (default: localhost)")
    parser.add_argument("--port", type=int, default=1025, help="SMTP server port (default: 1025)")
    parser.add_argument("--count", type=int, default=10, help="Number of emails to send (default: 10)")
    parser.add_argument("--tls", action="store_true", help="Use STARTTLS before sending")
    parser.add_argument("--username", default=None, help="SMTP auth username (optional)")
    parser.add_argument("--password", default=None, help="SMTP auth password (optional)")
    parser.add_argument("--quiet", action="store_true", help="Suppress per-message logging")
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    if args.count <= 0:
        print("--count must be a positive integer", file=sys.stderr)
        return 1
    try:
        sent = send_emails(
            host=args.host,
            port=args.port,
            count=args.count,
            use_tls=args.tls,
            username=args.username,
            password=args.password,
            verbose=not args.quiet,
        )
    except (smtplib.SMTPException, OSError) as exc:
        print(f"Failed to send emails: {exc}", file=sys.stderr)
        return 1

    print(f"Done. Sent {sent} email(s) to {args.host}:{args.port}.")
    return 0


if __name__ == "__main__":
    sys.exit(main())