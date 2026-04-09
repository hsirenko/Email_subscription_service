# RELEASE RADAR (subscribe UI)

Cyber-funk static UI for `POST /api/subscribe`. Deploy to **Vercel** with root directory **`web`**.

- Set **`VITE_API_URL`** to your Fly API base (e.g. `https://your-app.fly.dev`, no trailing slash).
- See the main repo **README** for Fly.io, CORS, and SMTP.

```bash
npm install
npm run dev    # http://localhost:5173 — API must allow this origin via CORS
npm run build
```
