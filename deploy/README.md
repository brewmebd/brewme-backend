# Server setup — BrewMe API

One-time setup on the Ubuntu server. After this, every push to `main` builds the
binary, copies it to `/var/www/brewme-backend/brewme`, and restarts it via Supervisor
(see [.github/workflows/deploy.yml](../.github/workflows/deploy.yml)).

## 1. Directories

```bash
sudo mkdir -p /var/www/brewme-backend/uploads/avatar
sudo chown -R www-data:www-data /var/www/brewme-backend
```

## 2. Environment file

The binary loads `.env` from its working directory, so it must live next to the binary.
It is gitignored and never shipped by CI — create it manually:

```bash
sudo -u www-data nano /var/www/brewme-backend/.env
# Fill in DATABASE_DSN, REDIS_*, JWT secret, etc. (see .env.example)
```

## 3. Supervisor

```bash
sudo apt install -y supervisor
sudo nano /etc/supervisor/conf.d/brewme-backend.conf
```

```ini
[program:brewme-backend]
command=/var/www/brewme-backend/brewme
directory=/var/www/brewme-backend
autostart=true
autorestart=true
user=www-data
stdout_logfile=/var/log/brewme-backend.out.log
stderr_logfile=/var/log/brewme-backend.err.log
environment=HOME="/var/www/brewme-backend"
```

> `directory=` is required — the app resolves `.env` and `uploads/` relative to it.

```bash
sudo supervisorctl reread
sudo supervisorctl update
sudo supervisorctl status brewme-backend
```

## 4. Nginx

```bash
sudo apt install -y nginx
sudo cp deploy/nginx.conf /etc/nginx/sites-available/brewme-backend
# edit server_name to your real domain
sudo ln -s /etc/nginx/sites-available/brewme-backend /etc/nginx/sites-enabled/
sudo nginx -t          # test config
sudo systemctl reload nginx
```

## 5. TLS (Let's Encrypt)

The nginx config references certbot cert paths, so issue the cert before reloading
with the 443 block enabled. Easiest path — let certbot patch nginx automatically:

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d api.brewme.example.com
sudo systemctl reload nginx
```

Certbot auto-renews via a systemd timer; verify with `sudo certbot renew --dry-run`.

> If you don't have a domain/TLS yet, comment out the `listen 443` server block and
> the `return 301` redirect, and let the `listen 80` block proxy to the app directly.

## 6. GitHub Actions secrets

Repo → Settings → Secrets and variables → Actions:

| Secret | Value |
|---|---|
| `SSH_HOST` | server IP / hostname |
| `SSH_USER` | deploy user (needs `supervisorctl` rights) |
| `SSH_PORT` | SSH port (e.g. 22) |
| `SSH_PRIVATE_KEY` | private key whose public half is in the server's `authorized_keys` |

> The deploy user must be able to run `supervisorctl` without a password prompt
> (add it to the `supervisor` group, or grant a targeted sudoers rule).
