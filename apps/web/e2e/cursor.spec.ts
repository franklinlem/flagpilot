import { expect, test } from '@playwright/test';
import { readFileSync } from 'node:fs';
test('botão sem alterações não indica carregamento', async ({ page }) => {
  await page.setContent('<button disabled>Salvar papel</button>');
  await page.addStyleTag({ content: readFileSync('apps/web/src/app/styles.css', 'utf8') });
  const button = page.getByRole('button', { name: 'Salvar papel' });
  await expect(button).toHaveCSS('cursor', 'default');
  await button.evaluate(element => element.setAttribute('aria-busy', 'true'));
  await expect(button).toHaveCSS('cursor', 'progress');
  await button.evaluate(element => element.setAttribute('aria-busy', 'false'));
  await expect(button).toHaveCSS('cursor', 'default');
});
