// @vitest-environment jsdom
import { afterEach, expect, it } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { EventDetails } from './EventDetails';
afterEach(cleanup);
it('exibe antes/depois, percentual e HTML como texto', () => {
  render(<EventDetails event={{before:null,after:{rolloutBps:3725,enabled:false,description:'<script>alert(1)</script>'}}} />);
  expect(screen.getByText('37.25%')).toBeTruthy(); expect(screen.getByText('Não')).toBeTruthy();
  expect(screen.getByText('<script>alert(1)</script>')).toBeTruthy(); expect(document.querySelector('script')).toBeNull();
  expect(screen.getByText('Sem campos disponíveis.')).toBeTruthy();
  expect(screen.getByText('Ver alterações').tagName).toBe('SUMMARY');
});
