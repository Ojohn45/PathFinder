# Putting PathFinder online

Three ways. Pick one, all end with `https://your-domain` serving the app.

- **Way 1: your own VPS** with Docker and Caddy. Cheapest long term, full control.
- **Way 2: Render** if you would rather not touch a server. See **RENDER.md** for the
  full walkthrough with `pathfinder.app`. Note that keeping your data needs the paid
  Starter plan, because a free Render service wipes its disk on every restart.
- **Way 3: other platforms.** Fly.io, Railway, and the same persistent-disk caveat.

---

## Way 1: your own server (about $5 a month, full control)

This is the one to use. You get a real machine, the JSON files sit on a real disk,
and nothing resets behind your back.

### 1. Buy a domain

Any registrar works: Cloudflare, Porkbun, Namecheap. Around $10 to $15 a year.

I checked these while writing this:

| domain | status |
|---|---|
| pathfinder.com | taken |
| pathfinder.app | free |
| pathfinder.dev | free |
| pathfinder.online | free |
| pathfinder.site | free |
| pathfindergo.com | free |

`.app` and `.dev` force HTTPS, which is a good thing, and Caddy handles it for you.

### 2. Rent a server

Hetzner (cheapest), DigitalOcean, Vultr or Linode. The smallest Ubuntu 24.04 box is
plenty: 1 vCPU, 1 GB of RAM. Note the server's public IP address.

### 3. Point the domain at the server

In your registrar's DNS panel add two records:

| type | name | value |
|---|---|---|
| A | @ | your server IP |
| A | www | your server IP |

Wait a few minutes. Check with `ping pathfinder.app` - it should answer with your IP.

### 4. Put the code on the server

```bash
ssh root@your-server-ip

# Docker, the only thing you need to install
curl -fsSL https://get.docker.com | sh

# upload the project, then:
cd /opt
git clone <your repo> pathfinder   # or scp the folder up
cd pathfinder
cp .env.example .env
nano .env
```

In `.env` set at least:

```
ADMIN_PASSWORD=a-long-password-you-choose
BASE_URL=https://pathfinder.app
SMTP_USER=your Gmail address
SMTP_PASS=the 16 letter Google App Password
```

`BASE_URL` matters: invitation emails use it to build their links, so it must be the
real domain, not localhost.

### 5. Put your domain in the Caddyfile

```bash
nano Caddyfile
```

```
pathfinder.app {
    encode zstd gzip
    reverse_proxy pathfinder:8080
}
```

### 6. Start it

```bash
docker compose up -d --build
```

That builds the app, starts it, starts Caddy, fetches the HTTPS certificate and
proxies the domain to the app. Open `https://pathfinder.app` - it should be live.

```bash
docker compose logs -f pathfinder   # watch it
docker compose restart pathfinder   # restart it
docker compose up -d --build        # after you change code
```

### Where the data lives

The `pathfinder-data` Docker volume holds `users.json`, `history.json` and
`invites.json`. It survives restarts and rebuilds. Back it up with:

```bash
docker run --rm -v pathfinder-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/pathfinder-backup.tar.gz -C /data .
```

### Without Docker

There is a `pathfinder.service` file for systemd. Build with `go build -o pathfinder .`,
put the folder in `/opt/pathfinder`, then:

```bash
sudo cp pathfinder.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now pathfinder
```

You would still need something in front for HTTPS. Install Caddy
(`apt install caddy`) and use the same Caddyfile, pointing at `localhost:8080`.

---

## Way 2: a hosting platform (free to start, one catch)

Render, Railway and Fly.io all run a Go app from a Git repo. Push the repo, they
build it, you get a URL, then add your domain in their dashboard.

The catch is the storage. This app keeps its data in JSON files on disk. On the free
tiers the disk is thrown away on every deploy and restart, so every signup and every
visit history would vanish. You must attach a persistent disk:

- **Fly.io**: `fly volumes create pathfinder_data`, then mount it at `/app/storage`.
- **Railway**: add a volume, mount path `/app/storage`.
- **Render**: a persistent disk is a paid feature, mount it at `/app/storage`.

Also set the environment variables from `.env` in their dashboard (`ADMIN_PASSWORD`,
`BASE_URL`, `SMTP_USER`, `SMTP_PASS`). They pass a port in `$PORT`, which the app now
reads, so no code change is needed.

If you would rather not think about disks, use Way 1.

---

## After it is live

1. Log in at `https://your-domain/admin/login` as `Ken` and change the password:
   put a new `ADMIN_PASSWORD` in `.env`, then `docker compose up -d` restarts and
   updates it.
2. Open Admin, then Emails, and send yourself a test email. If it arrives, invitations
   and welcome mails work.
3. Invite one friend and follow the link yourself to be sure the flow is right.
4. Keep the `storage` volume backed up. That folder is the whole app's memory.
