import { PPTXGenerator } from '../src/pptx/generator';

describe('PPTXGenerator', () => {
  it('should create a generator instance', () => {
    const gen = new PPTXGenerator({ title: 'Test' });
    expect(gen).toBeDefined();
  });

  it('should add slides without errors', () => {
    const gen = new PPTXGenerator({ title: 'Test Presentation' });
    gen.addSlide({
      title: 'Slide 1',
      content: [{ text: 'Hello World' }],
      notes: 'Speaker notes here',
    });
    gen.addSlide({
      content: [
        { text: 'Bullet 1', bold: true },
        { text: 'Bullet 2', italic: true },
      ],
    });
    expect(gen).toBeDefined();
  });

  it('should generate buffer', async () => {
    const gen = new PPTXGenerator({ title: 'Buffer Test' });
    gen.addSlide({
      title: 'Test',
      content: [{ text: 'Content' }],
    });
    const buffer = await gen.toBuffer();
    expect(buffer).toBeDefined();
    expect(buffer.length).toBeGreaterThan(0);
  });
});
