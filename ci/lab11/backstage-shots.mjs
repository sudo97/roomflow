// Знімки порталу Backstage (вхід як guest).
import { createRequire } from 'module';
const require = createRequire('/tmp/pw/package.json');
const { chromium } = require('playwright');
const out = process.argv[2];
const base = 'http://localhost:3000';
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
await page.goto(base, { waitUntil: 'networkidle', timeout: 120000 });
for (const label of ['Enter', 'Sign In', 'Continue']) {
  const b = page.getByRole('button', { name: label });
  if (await b.count()) { await b.first().click(); break; }
}
await page.waitForTimeout(4000);
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
    await page.waitForTimeout(3000);
    await page.screenshot({ path: `${out}/${name}.png` });
    console.log('saved', name);
  } catch (e) { console.log('skip', name, e.message); }
}
await browser.close();
