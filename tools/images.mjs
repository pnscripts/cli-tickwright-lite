// README / Docker Hub / GitHub images from real output (tools/terminal.txt, made by the binary).
import fs from 'node:fs';
import { slide, banner, esc } from './brand.mjs';
const out = fs.readFileSync(new URL('./terminal.txt', import.meta.url), 'utf8');
export default [
  { out: 'listing/terminal-1280x720.png', width: 1280, height: 720, html: slide(1280, 720, { kicker: 'Tickwright Lite', title: 'Cron, explained, with the real next runs', body: `<span style="font-size:17px">${esc(out)}</span>` }) },
  { out: 'listing/social-preview-1280x640.png', width: 1280, height: 640, html: banner(1280, 640, { name: 'Tickwright Lite', tagline: 'Validate and explain cron expressions and see their next runs in any time zone.', chips: ['validate', 'explain EN/BG', 'next runs', 'DST-correct', 'Docker'], glyph: '⏱', kicker: 'Free CLI · MIT' }) }
];
