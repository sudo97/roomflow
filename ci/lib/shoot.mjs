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
if (process.env.SHOT_LOGIN_URL) { // вхід через форму: перше текстове поле — логін, поле password — пароль
  try {
    await page.goto(process.env.SHOT_LOGIN_URL, { waitUntil: 'networkidle', timeout: 60000 });
    await page.fill('input[type=text], input[name=username]', process.env.SHOT_USER || '');
    await page.fill('input[type=password]', process.env.SHOT_PASS || '');
    const btn = page.getByRole('button', { name: /sign in/i });
    if (await btn.count()) await btn.first().click(); else await page.keyboard.press('Enter');
    await page.waitForURL(u => !String(u).includes('/login'), { timeout: 20000 }).catch(() => console.log('login: still on login page'));
    await page.waitForTimeout(2000);
  } catch (e) { console.log('login:', e.message); }
}
try {
  await page.goto(url, { waitUntil: 'networkidle', timeout: 90000 });
} catch (e) { console.log('goto:', e.message); }
await page.waitForTimeout(+wait);
await page.screenshot({ path: out });
console.log('saved', out);
await browser.close();
