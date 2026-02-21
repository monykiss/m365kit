#!/usr/bin/env node

/**
 * CLI bridge for M365Kit Go→Node subprocess communication.
 * Reads a JSON request from stdin, performs the action, writes JSON response to stdout.
 *
 * Protocol:
 *   stdin:  { "action": "pptx.generate", "output": "/path/out.pptx", "options": {...}, "slides": [...] }
 *   stdout: { "success": true, "path": "/path/out.pptx" }  or  { "error": "message" }
 */

import { PPTXGenerator, SlideData, PPTXOptions } from './pptx/generator';

interface GenerateRequest {
  action: 'pptx.generate';
  output: string;
  options?: PPTXOptions;
  slides: SlideData[];
}

type Request = GenerateRequest;

async function main(): Promise<void> {
  const chunks: Buffer[] = [];
  for await (const chunk of process.stdin) {
    chunks.push(chunk);
  }
  const input = Buffer.concat(chunks).toString('utf-8');

  let req: Request;
  try {
    req = JSON.parse(input);
  } catch {
    respond({ error: 'invalid JSON input' });
    return;
  }

  if (!req.action) {
    respond({ error: 'missing "action" field' });
    return;
  }

  switch (req.action) {
    case 'pptx.generate':
      await handlePPTXGenerate(req);
      break;
    default:
      respond({ error: `unknown action: ${req.action}` });
  }
}

async function handlePPTXGenerate(req: GenerateRequest): Promise<void> {
  if (!req.output) {
    respond({ error: 'missing "output" field' });
    return;
  }
  if (!req.slides || !Array.isArray(req.slides) || req.slides.length === 0) {
    respond({ error: 'missing or empty "slides" array' });
    return;
  }

  const gen = new PPTXGenerator(req.options ?? {});
  gen.addSlides(req.slides);
  await gen.writeFile(req.output);
  respond({ success: true, path: req.output, slidesCount: req.slides.length });
}

function respond(data: Record<string, unknown>): void {
  process.stdout.write(JSON.stringify(data) + '\n');
}

main().catch((err) => {
  respond({ error: String(err?.message ?? err) });
  process.exit(1);
});
