import PptxGenJS from 'pptxgenjs';

export interface SlideContent {
  text: string;
  fontSize?: number;
  bold?: boolean;
  italic?: boolean;
  color?: string;
  x?: number;
  y?: number;
  w?: number;
  h?: number;
}

export interface SlideData {
  title?: string;
  content: SlideContent[];
  notes?: string;
  layout?: 'TITLE' | 'TITLE_ONLY' | 'BLANK' | 'SECTION';
}

export interface PPTXOptions {
  title?: string;
  author?: string;
  subject?: string;
  company?: string;
}

export class PPTXGenerator {
  private pptx: PptxGenJS;

  constructor(options: PPTXOptions = {}) {
    this.pptx = new PptxGenJS();
    if (options.title) this.pptx.title = options.title;
    if (options.author) this.pptx.author = options.author;
    if (options.subject) this.pptx.subject = options.subject;
    if (options.company) this.pptx.company = options.company;
  }

  addSlide(data: SlideData): void {
    const slide = this.pptx.addSlide();

    if (data.title) {
      slide.addText(data.title, {
        x: 0.5,
        y: 0.5,
        w: '90%',
        h: 1,
        fontSize: 28,
        bold: true,
        color: '363636',
      });
    }

    for (const content of data.content) {
      slide.addText(content.text, {
        x: content.x ?? 0.5,
        y: content.y ?? (data.title ? 2 : 0.5),
        w: content.w ?? 9,
        h: content.h ?? 4,
        fontSize: content.fontSize ?? 18,
        bold: content.bold ?? false,
        italic: content.italic ?? false,
        color: content.color ?? '363636',
      });
    }

    if (data.notes) {
      slide.addNotes(data.notes);
    }
  }

  addSlides(slides: SlideData[]): void {
    for (const slide of slides) {
      this.addSlide(slide);
    }
  }

  async writeFile(path: string): Promise<void> {
    await this.pptx.writeFile({ fileName: path });
  }

  async toBuffer(): Promise<Buffer> {
    const data = await this.pptx.write({ outputType: 'nodebuffer' });
    return data as Buffer;
  }
}
