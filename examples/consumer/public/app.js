const form = document.querySelector('#quote');
const result = document.querySelector('#result');
const button = form.querySelector('button');
const money = value => new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(value / 100);
form.addEventListener('submit', async event => {
  event.preventDefault();
  button.disabled = true;
  result.textContent = 'Calculando…';
  try {
    const response = await fetch('/api/quote', { method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ quantity: Number(form.elements.quantity.value) }), signal: AbortSignal.timeout(5000) });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error);
    result.textContent = `Experiência ${data.experience === 'new' ? 'nova' : 'padrão'}: produtos ${money(data.subtotalCents)}, frete ${money(data.shippingCents)}, total ${money(data.totalCents)}.${data.fallback ? ' Serviço de flags indisponível: experiência padrão preservada.' : ''}`;
  } catch {
    result.textContent = 'Não foi possível calcular. Confira a quantidade e tente novamente.';
  } finally { button.disabled = false; }
});
