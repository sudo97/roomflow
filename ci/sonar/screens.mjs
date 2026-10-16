// Знімки екрана веб-інтерфейсу SonarQube для звіту (Playwright).
import { chromium } from 'playwright';
const [,, outDir] = process.argv;
const base = process.env.SONAR_HOST_URL, pw = process.env.SONAR_ADMIN_PASSWORD;
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, locale: 'en-US' });
await page.goto(`${base}/sessions/new`);
await page.fill('#login', 'admin');
await page.fill('#password', pw);
await page.click('button[type=submit]');
await page.waitForTimeout(4000);
const shots = {
  '01-dashboard': '/dashboard?id=roomflow',
  '02-issues': '/project/issues?id=roomflow&resolved=false',
  '03-measures': '/component_measures?id=roomflow',
  '04-hotspots': '/security_hotspots?id=roomflow',
  '05-quality-gate': '/quality_gates',
  '06-profiles': '/profiles?language=go',
  '07-activity': '/project/activity?id=roomflow&graph=issues',
  '08-code': '/code?id=roomflow',
};
for (const [name, path] of Object.entries(shots)) {
  try {
    await page.goto(base + path, { waitUntil: 'networkidle', timeout: 60000 });
    await page.waitForTimeout(2500);
    await page.screenshot({ path: `${outDir}/${name}.png`, fullPage: false });
    console.log('saved', name);
  } catch (e) { console.log('skip', name, e.message); }
}
await browser.close();
