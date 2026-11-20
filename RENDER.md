# PathFinder on Render, at pathfinder.app

Render runs the app from a GitHub repo and gives you HTTPS automatically. There is one
thing to know before you start.

**The free plan cannot keep your data.** A free Render web service has an ephemeral
disk: every redeploy, restart, or 15 minutes of no visitors wipes `users.json`,
`history.json` and `invites.json`. Every signup and every bit of reading history would
disappear. A persistent disk is only available on paid instance types, so use the
**Starter** plan (about $7 a month) plus a 1 GB disk (about $0.25 a month). The
`render.yaml` in this folder already asks for exactly that.

---

## 1. Get the code onto GitHub

```bash
cd pathfinder
git init
git add .
git commit -m "PathFinder"
git branch -M main
git remote add origin https://github.com/YOUR-NAME/pathfinder.git
git push -u origin main
```

Check that `.gitignore` kept `.env` out of the repo. It is already listed there.

## 2. Create the service

1. Sign up at render.com and connect GitHub.
2. **New > Blueprint**, choose the `pathfinder` repo, click **Apply**.
3. Render reads `render.yaml`: one Docker web service, Starter plan, Frankfurt, and a
   1 GB disk mounted at `/app/storage`.
4. It asks you for the two secret values. Fill them in:
   - `ADMIN_PASSWORD`: the password you want for Ken.
   - `SMTP_USER` and `SMTP_PASS`: your Gmail address and the 16 letter Google App Password.
5. Wait for the first build. You get a URL like `pathfinder-xxxx.onrender.com`. Open it.

If you would rather not use a Blueprint, create a **New > Web Service**, pick the repo,
choose **Docker** as the environment, set the plan to Starter, then add the same
environment variables by hand and add the disk with mount path `/app/storage`.

## 3. Point pathfinder.app at it

Buy the domain first (Cloudflare, Porkbun, Namecheap). Then:

1. In Render, open the service, **Settings**, scroll to **Custom Domains**,
   **+ Add Custom Domain**, and enter `pathfinder.app`.
2. Render also lists `www.pathfinder.app`. Keep both.
3. Add these DNS records at your registrar:

   | type | name | value |
   |---|---|---|
   | A | @ | 216.24.57.1 |
   | CNAME | www | your-service.onrender.com |

   `216.24.57.1` is Render's load balancer. If your registrar supports ALIAS or ANAME,
   use that instead of the A record and point it at `your-service.onrender.com`.
   If you use Cloudflare, set the record to **DNS only** (grey cloud) while Render
   verifies the domain, and delete any AAAA records.
4. Back in Render, click **Verify**. It issues the TLS certificate, usually within a
   few minutes. Then `https://pathfinder.app` serves the app.

## 4. BASE_URL (optional)

Invitation emails build their links from `BASE_URL`. You do **not** need a domain for
this: if `BASE_URL` is empty, PathFinder falls back to Render's own
`RENDER_EXTERNAL_URL`, which is the service's `https://<service>.onrender.com` address.
So invites work on day one, before any domain exists.

Once `pathfinder.app` is live and verified, set `BASE_URL=https://pathfinder.app` in the
service's Environment tab and redeploy, so new invite links use the pretty domain.
Old links keep working either way.

## 4b. Email without touching variables

You can skip the SMTP variables entirely. Once the site is up, log in as Ken, open
**Admin > Email Settings**, type the Gmail address and its 16 letter App Password, and
press **Save and send a test email**. PathFinder keeps the details in
`storage/settings.json` on the disk, so they survive restarts. Environment variables,
if you do set them, take priority over the page.

## 4c. Admin sign-in

No admin page is needed. Log in at `/login` with the registered number (08127758529)
and the password, and you land on the dashboard. Set `ADMIN_PHONE` if the number ever
changes, and `ADMIN_PASSWORD` to set the password on first boot.

## 5. After it is live

- Log in at `https://pathfinder.app/admin/login` as `Ken`.
- Admin > Emails > send yourself a test. If it arrives, welcome and invitation mail work.
- Admin > Invite a friend, follow the link yourself, make sure the flow is right.
- Free plan note: if you ever downgrade to Free, the disk detaches and the data is gone.
- Backups: from your own machine you can pull the three JSON files down any time with
  Render's shell, or copy them from the disk using the dashboard's shell tab.

## Costs, plainly

| item | monthly |
|---|---|
| Starter web service | about $7 |
| 1 GB persistent disk | about $0.25 |
| domain, spread monthly | about $1 |
| **total** | **about $8** |

Render's free plan is still useful for a look around, but treat it as a demo: signups
and history will reset.
