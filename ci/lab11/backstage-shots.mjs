// Знімки порталу Backstage (вхід як guest).
import { createRequire } from 'module';
const require = createRequire('/tmp/pw/package.json');
const { chromium } = require('playwright');
const out = process.argv[2];
const base = 'http://localhost:3000';
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
// Вхід як guest: кнопка Enter з'являється після завантаження SPA, тому її чекаємо явно;
// перевіряємо і після кожного переходу (повне перезавантаження сторінки може знову показати вхід).
async function signIn() {
  for (const label of ['Enter', 'Sign In', 'Continue']) {
    const b = page.getByRole('button', { name: label });
    try { await b.first().waitFor({ state: 'visible', timeout: 8000 }); await b.first().click(); await page.waitForTimeout(3000); return true; } catch { /* немає такої кнопки */ }
  }
  return false;
}
await page.goto(base, { waitUntil: 'networkidle', timeout: 120000 });
console.log('signed in:', await signIn());
const shots = {
  'catalog': '/catalog?filters[kind]=component',
  'component-api': '/catalog/default/component/roomflow-api',
  'dependencies': '/catalog/default/component/roomflow-api/dependencies',
  'system': '/catalog/default/system/roomflow',
  'api-docs': '/catalog/default/api/roomflow-api/definition',
  'templates': '/create',
  'template-form': '/create/templates/default/roomflow-service',
};
for (const [name, path] of Object.entries(shots)) {
  try {
    await page.goto(base + path, { waitUntil: 'networkidle', timeout: 60000 });
    if (await page.getByRole('button', { name: 'Enter' }).count()) await signIn();
    await page.waitForTimeout(3000);
    await page.screenshot({ path: `${out}/${name}.png` });
    console.log('saved', name);
  } catch (e) { console.log('skip', name, e.message); }
}
await browser.close();
