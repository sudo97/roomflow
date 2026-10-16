// Знімок екрана сторінки для звіту: node shoot.mjs out.png url [wait_ms] [width] [height] [cookie=value]
import { createRequire } from 'module';
const require = createRequire('/tmp/pw/package.json');
const { chromium } = require('playwright');
const [,, out, url, wait = '3000', width = '1440', height = '900'] = process.argv;
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: +width, height: +height }, locale: 'en-US', ignoreHTTPSErrors: true });
if (process.env.SHOT_AUTH) await page.setExtraHTTPHeaders({ Authorization: process.env.SHOT_AUTH });
if (process.env.SHOT_COOKIE) { // name=value
  const [name, ...v] = process.env.SHOT_COOKIE.split('=');
  await page.context().addCookies([{ name, value: v.join('='), url: new URL(url).origin }]);
}
try {
  await page.goto(url, { waitUntil: 'networkidle', timeout: 90000 });
} catch (e) { console.log('goto:', e.message); }
await page.waitForTimeout(+wait);
await page.screenshot({ path: out });
console.log('saved', out);
await browser.close();
