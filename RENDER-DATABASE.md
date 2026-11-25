# Keeping accounts forever, and running PathFinder on Render

## Why accounts were disappearing

PathFinder saved everything in `storage/*.json` (users, history, invites, the email
log, mail settings). On Render that folder is **wiped on every redeploy, every
restart and after the service sleeps**. The logout button was innocent: it only
deletes the session cookie and never touches the data files.

The fix is to keep the data somewhere that is not the server's own disk: Postgres.

## 1. Create a free Postgres database (Neon)

1. Go to https://neon.com and sign up (GitHub sign-in is fastest).
2. Create a project. Pick the region closest to you; `eu-central-1` (Frankfurt)
   is the closest to Nigeria among the usual choices.
3. Copy the **connection string**. It looks like:
   `postgresql://user:password@ep-xxx.eu-central-1.aws.neon.tech/neondb?sslmode=require`

Neon's free plan: 0.5 GB storage, no expiry date, and the database **suspends
after about 5 minutes of no activity**. The next request wakes it in under a
second. Nothing is deleted.

Alternatives, honestly:

| Option | Free tier | Catch |
|---|---|---|
| Neon | 0.5 GB, no expiry | sleeps after ~5 min idle, wakes on next request |
| Supabase | 500 MB | project **pauses after 7 days** of no use; you must click to resume |
| Render Postgres | free for **30 days only** | deleted after 30 days unless you pay |
| Railway / Aiven | small free/trial credit | runs out |

Neon is the one I recommend.

## 2. Tell Render about it

Render dashboard, your PathFinder service, **Environment**, add:

| Key | Value |
|---|---|
| `DATABASE_URL` | the Neon connection string |
| `ADMIN_PASSWORD` | your head admin password |
| `ADMIN_USERNAME` | `Ken` (optional, this is the default) |
| `ADMIN_PHONE` | `08127758529` (optional) |
| `BASE_URL` | `https://pathfinder-1q2y.onrender.com` (or your domain) |
| `SMTP_USER` | your Gmail address (optional, can also be set in Admin > Email Settings) |
| `SMTP_PASS` | the Gmail app password (optional) |

Never put the connection string in the code. The app reads it from the
environment only.

## 3. Build and start commands

`render.yaml` already sets these; if you created the service by hand, use:

- Build: `go build -o pathfinder .`
- Start: `./pathfinder`

## 4. Redeploy

Push to GitHub. Render redeploys automatically. On the first start you should see
in the logs:

```
STORAGE: using Postgres. Accounts, history and settings survive redeploys.
Storage ready: N user(s), M history entries
```

If you see `DATABASE_URL not set`, the variable is missing or the service was not
redeployed after you added it.

## 5. What is where now

| Thing | Where it lives |
|---|---|
| Accounts, hashed passwords, roles | Postgres `users` table |
| Visit and search history | Postgres `history` table |
| Invitations | Postgres `invites` table |
| Email log | Postgres `email_log` table |
| Mail settings and tab icon | Postgres `settings` table |
| Logins that survive a restart | Postgres `sessions` table |
| The 71 languages and every place | `content.go` and `data.go`, compiled into the program. No database needed. |

Tables are created automatically on first start. Running it again changes nothing.

## 6. Honest limits

- Accounts that were already lost cannot be recovered. They were on a disk that
  Render had already wiped; nothing on the server kept a copy.
- The free Neon database sleeps when unused. The first page load after a quiet
  period takes about a second longer.
- Back up occasionally: in Neon, **Branches > Create branch** makes an instant copy.
