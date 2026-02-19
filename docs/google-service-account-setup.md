# Google Service Account Setup

This guide walks through creating a Google Cloud service account for the Calendar API.

## Step 1: Create a Google Cloud Project

1. Open [Google Cloud Console](https://console.cloud.google.com/)
2. Click the project dropdown → **New Project**
3. Enter a name (e.g., `calendar-mcp`) → **Create**

## Step 2: Enable the Calendar API

1. Go to **APIs & Services** → **Library**
2. Search for **Google Calendar API**
3. Click **Enable**

## Step 3: Create a Service Account

1. Go to **APIs & Services** → **Credentials**
2. Click **Create Credentials** → **Service account**
3. Enter a name (e.g., `calendar-mcp`) → **Create and Continue**
4. Skip the optional roles → **Continue** → **Done**

## Step 4: Create a JSON Key

1. In the Service Accounts list, click on the account you created
2. Go to the **Keys** tab
3. Click **Add Key** → **Create new key** → **JSON** → **Create**
4. The key file downloads automatically — save it securely

The file looks like:

```json
{
  "type": "service_account",
  "project_id": "your-project-id",
  "private_key_id": "...",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "client_email": "calendar-mcp@your-project.iam.gserviceaccount.com",
  ...
}
```

Note the `client_email` — you'll need it in the next step.

## Step 5: Share Your Calendar

The service account has no calendar access by default. You must share your calendar with it.

1. Open [Google Calendar](https://calendar.google.com/)
2. Find your calendar in the left panel → click **⋮** → **Settings and sharing**
3. Scroll to **Share with specific people or groups**
4. Click **Add people and groups**
5. Paste the `client_email` from the JSON key
6. Set permission to **Make changes to events** (for read/write access)
7. Click **Send**

## Step 6: Find Your Calendar ID

- **Primary calendar**: your Gmail address (e.g., `user@gmail.com`)
- **Other calendars**: Settings → **Integrate calendar** → **Calendar ID**

## Troubleshooting

**"Not Found" error** — calendar is not shared with the service account email, or the Calendar ID is wrong.

**"Insufficient Permission" error** — the service account was shared with read-only access. Use "Make changes to events" for full access.

**"The caller does not have permission"** — the Calendar API is not enabled in the project.
