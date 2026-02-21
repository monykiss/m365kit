import {
  Document,
  Paragraph,
  TextRun,
  HeadingLevel,
  Packer,
  AlignmentType,
} from 'docx';
import * as fs from 'fs';

export interface ParagraphData {
  text: string;
  heading?: 1 | 2 | 3 | 4 | 5;
  bold?: boolean;
  italic?: boolean;
  alignment?: 'left' | 'center' | 'right' | 'justify';
}

export interface DocumentData {
  title?: string;
  author?: string;
  paragraphs: ParagraphData[];
}

const headingMap: Record<number, (typeof HeadingLevel)[keyof typeof HeadingLevel]> = {
  1: HeadingLevel.HEADING_1,
  2: HeadingLevel.HEADING_2,
  3: HeadingLevel.HEADING_3,
  4: HeadingLevel.HEADING_4,
  5: HeadingLevel.HEADING_5,
};

const alignmentMap: Record<string, (typeof AlignmentType)[keyof typeof AlignmentType]> = {
  left: AlignmentType.LEFT,
  center: AlignmentType.CENTER,
  right: AlignmentType.RIGHT,
  justify: AlignmentType.JUSTIFIED,
};

export class DOCXGenerator {
  private data: DocumentData;

  constructor(data: DocumentData) {
    this.data = data;
  }

  private buildParagraphs(): Paragraph[] {
    return this.data.paragraphs.map((p) => {
      const run = new TextRun({
        text: p.text,
        bold: p.bold,
        italics: p.italic,
      });

      const options: ConstructorParameters<typeof Paragraph>[0] = {
        children: [run],
      };

      if (p.heading && headingMap[p.heading]) {
        options.heading = headingMap[p.heading];
      }

      if (p.alignment && alignmentMap[p.alignment]) {
        options.alignment = alignmentMap[p.alignment];
      }

      return new Paragraph(options);
    });
  }

  async toBuffer(): Promise<Buffer> {
    const doc = new Document({
      creator: this.data.author ?? 'M365Kit',
      title: this.data.title,
      sections: [
        {
          children: this.buildParagraphs(),
        },
      ],
    });

    return await Packer.toBuffer(doc);
  }

  async writeFile(path: string): Promise<void> {
    const buffer = await this.toBuffer();
    fs.writeFileSync(path, buffer);
  }
}
