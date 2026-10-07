const https = require('https');
const http = require('http');
const fs = require('fs');
const spawn = require('child_process').spawn;
const express = require('express');
const { createProxyMiddleware } = require('http-proxy-middleware');

// Belt-and-suspenders for any residual BE chrome if an older build is still cached.
// Primary removal is in-tree (Sidebar / Footer / system version handler).
const INJECTED_HTML = `
  <style>
    /* Upgrade to Business Edition button (top of sidebar) */
    .sidebar > button,
    button:has(> .lucide):has(+ *),
    .sidebar button[class*="bg-[#023959]"],
    .sidebar button[class*="bg-\\[\\#023959\\]"] {
      display: none !important;
    }

    /* BE limited feature overlays */
    .be-indicator-container,
    .limited-be {
      display: none !important;
    }

    .oauth-save-settings-button {
      display: inline-block !important;
    }

    .be-indicator {
      filter: saturate(0) !important;
      opacity: 0.2 !important;
      pointer-events: none !important;
    }

  </style>
  <script>
    (function () {
      var headNode = document.getElementsByTagName('script')[0];
      if (!headNode || !headNode.parentNode) return;
      headNode = headNode.parentNode;
      headNode.originalInsertBefore = headNode.insertBefore;
      headNode.insertBefore = function(newNode, referenceNode) {
        if (newNode && newNode.src && newNode.src.indexOf('matomo') !== -1) {
          console.log('Blocked insertion of matomo script node');
        } else {
          headNode.originalInsertBefore(newNode, referenceNode);
        }
      };

      // Hide Upgrade BE button by text (in-tree removal is primary; this is a safety net)
      function hideUpgradeBe() {
        document.querySelectorAll('.sidebar button, nav button, button').forEach(function (btn) {
          var t = (btn.textContent || '').trim();
          if (t.indexOf('Upgrade to Business') !== -1) {
            btn.style.setProperty('display', 'none', 'important');
          }
        });
      }
      if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', hideUpgradeBe);
      } else {
        hideUpgradeBe();
      }
      setInterval(hideUpgradeBe, 2000);
    })();
  </script>
`;
const TARGET_URL = 'http://localhost:19000';
const SSL_CERT_PATH = '/data/certs/cert.pem';
const SSL_KEY_PATH = '/data/certs/key.pem';
const FORWARDED_ARGS = process.argv.slice(2);

const app = express();
app.get('/', async (req, res) => {
  try {
    const response = await fetch(TARGET_URL);
    const body = await response.text();
    const newBody = body.replace('<head>', `<head>${INJECTED_HTML}`);
    res.send(newBody);
  } catch (e) {
    console.error(e);
    res.status(500).json(e);
  }
});
app.get('/api/motd', (req, res) => {
  res.json({});
});
app.use(createProxyMiddleware({
  target: TARGET_URL,
  ws: true,
}));

async function waitUntilCertAvailable() {
  const sleep = (ms) => new Promise(r => setTimeout(r, ms));
  while (!fs.existsSync(SSL_CERT_PATH)) {
    await sleep(1000);
  }
}
async function runServer() {
  http.createServer(app).listen(9000);
  await waitUntilCertAvailable();
  https.createServer({
    key: fs.readFileSync(SSL_KEY_PATH),
    cert: fs.readFileSync(SSL_CERT_PATH),
  }, app).listen(9443);
}

function runPortainer() {
  const fwdArgs = FORWARDED_ARGS.join(' ');
  console.log(`Launching portainer with args ${fwdArgs}`)
  const child = spawn('/bin/sh', [
    '-c',
    `/portainer --bind=":19000" --bind-https=":19443" ${fwdArgs}`
  ]);
  child.stdout.pipe(process.stdout);
  child.stderr.pipe(process.stderr);
  child.on('exit', function (code) {
    console.log(`portainer exited with status code ${code}`);
    process.exit(code);
  });
  process.on('SIGTERM', function () {
    child.kill('SIGTERM');
  });
}

runPortainer();
runServer();
