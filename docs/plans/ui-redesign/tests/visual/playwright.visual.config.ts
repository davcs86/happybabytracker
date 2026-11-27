import { defineConfig } from '@playwright/test';
// Copy next to the repo's playwright.config.ts (reuse its webServer / stub API settings), and point snapshotPathTemplate at the baselines.
const base = { deviceScaleFactor: 1, locale: 'en-US', timezoneId: 'America/Los_Angeles' };
export default defineConfig({
  testDir: 'tests/visual',
  testMatch: '**/*.visual.spec.ts',
  // Baselines live in tests/visual/__baselines__/<variant>/<page>/NN-<state>.png : run tools/install-baselines.sh once to copy them from screenshots/.
  snapshotPathTemplate: '{testDir}/__baselines__/{projectName}/{arg}{ext}',
  workers: 1,
  projects: [
    { name: 'phone-light',   use: { ...base, viewport: { width: 390, height: 800 }, colorScheme: 'light' } },
    { name: 'phone-dark',    use: { ...base, viewport: { width: 390, height: 800 }, colorScheme: 'dark' } },
    { name: 'desktop-light', use: { ...base, viewport: { width: 980, height: 780 }, colorScheme: 'light' } },
    { name: 'desktop-dark',  use: { ...base, viewport: { width: 980, height: 780 }, colorScheme: 'dark' } },
    { name: 'wide-light',    use: { ...base, viewport: { width: 1280, height: 780 }, colorScheme: 'light' } },
    { name: 'wide-dark',     use: { ...base, viewport: { width: 1280, height: 780 }, colorScheme: 'dark' } },
  ],
});
