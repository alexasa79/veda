package editor

import (
	"bytes"
	"encoding/gob"
)

func encodeSessionValues(values ...interface{}) ([]byte, error) {
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	for _, value := range values {
		if err := enc.Encode(value); err != nil {
			return nil, err
		}
	}
	return data.Bytes(), nil
}
func decodeSessionValues(data []byte, values ...interface{}) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	for _, value := range values {
		if err := dec.Decode(value); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	gob.Register(&PasteCommand{})
	gob.Register(&RewrapHistoryEntry{})
	gob.Register(&markdownTableHistoryEntry{})
	gob.Register(DeleteOperator{})
	gob.Register(ChangeOperator{})
	gob.Register(YankOperator{})
	gob.Register(LineTextObject{})
	gob.Register(WholeBufferTextObject{})
	gob.Register(EndOfLineTextObject{})
	gob.Register(CharacterTextObject{})
	gob.Register(LineMotionTextObject{})
	gob.Register(MultiLineTextObject{})
	gob.Register(WordTextObject{})
	gob.Register(WTextObject{})
	gob.Register(BracketTextObject{})
	gob.Register(FindCharacterTextObject{})
	gob.Register(TillCharacterTextObject{})
	gob.Register(SearchResultTextObject{})
	gob.Register(ArgumentTextObject{})
	gob.Register(BlockTextObject{})
	gob.Register(FunctionTextObject{})
}

func (p *PasteCommand) GobEncode() ([]byte, error) {
	return encodeSessionValues(p.position, p.content, p.isLinewise, p.oldCursorX, p.oldCursorY)
}
func (p *PasteCommand) GobDecode(data []byte) error {
	return decodeSessionValues(data, &p.position, &p.content, &p.isLinewise, &p.oldCursorX, &p.oldCursorY)
}
func (r *RewrapHistoryEntry) GobEncode() ([]byte, error) {
	return encodeSessionValues(r.startLine, r.endLine, r.oldLines, r.newLines, r.oldCursorX, r.oldCursorY, r.startPos, r.endPos, r.newContent)
}
func (r *RewrapHistoryEntry) GobDecode(data []byte) error {
	return decodeSessionValues(data, &r.startLine, &r.endLine, &r.oldLines, &r.newLines, &r.oldCursorX, &r.oldCursorY, &r.startPos, &r.endPos, &r.newContent)
}
func (m *markdownTableHistoryEntry) GobEncode() ([]byte, error) {
	return encodeSessionValues(m.startLine, m.oldLines, m.newLines, m.oldCursorX, m.oldCursorY, m.newCursorX, m.newCursorY)
}
func (m *markdownTableHistoryEntry) GobDecode(data []byte) error {
	return decodeSessionValues(data, &m.startLine, &m.oldLines, &m.newLines, &m.oldCursorX, &m.oldCursorY, &m.newCursorX, &m.newCursorY)
}
func (n NavigationEntry) GobEncode() ([]byte, error) {
	return encodeSessionValues(n.filePath, n.fallbackFilePath, n.cursorX, n.cursorLineNumber, n.topLineNumber, n.fallbackCursorLine, n.fallbackTopLineNumber)
}
func (n *NavigationEntry) GobDecode(data []byte) error {
	return decodeSessionValues(data, &n.filePath, &n.fallbackFilePath, &n.cursorX, &n.cursorLineNumber, &n.topLineNumber, &n.fallbackCursorLine, &n.fallbackTopLineNumber)
}

func (t *TerminalEmulator) GobEncode() ([]byte, error) {
	return encodeSessionValues(t.width, t.height, t.curX, t.curY, t.cells, t.wrapPending, t.savedX, t.savedY, t.fg, t.bg, t.attrs, t.curFg, t.curBg, t.curAttrs, t.AppCursorKeys, t.topMargin, t.bottomMargin, t.scrollback, t.maxScrollback)
}
func (t *TerminalEmulator) GobDecode(data []byte) error {
	return decodeSessionValues(data, &t.width, &t.height, &t.curX, &t.curY, &t.cells, &t.wrapPending, &t.savedX, &t.savedY, &t.fg, &t.bg, &t.attrs, &t.curFg, &t.curBg, &t.curAttrs, &t.AppCursorKeys, &t.topMargin, &t.bottomMargin, &t.scrollback, &t.maxScrollback)
}
func (l termLine) GobEncode() ([]byte, error) {
	return encodeSessionValues(l.cells, l.fg, l.bg, l.attrs)
}
func (l *termLine) GobDecode(data []byte) error {
	return decodeSessionValues(data, &l.cells, &l.fg, &l.bg, &l.attrs)
}

func (w WordTextObject) GobEncode() ([]byte, error) {
	return encodeSessionValues(w.modifier, w.excludeTrailingSpace, w.Count)
}
func (w *WordTextObject) GobDecode(data []byte) error {
	return decodeSessionValues(data, &w.modifier, &w.excludeTrailingSpace, &w.Count)
}
func (b BracketTextObject) GobEncode() ([]byte, error) {
	return encodeSessionValues(b.openBracket, b.closeBracket, b.modifier)
}
func (b *BracketTextObject) GobDecode(data []byte) error {
	return decodeSessionValues(data, &b.openBracket, &b.closeBracket, &b.modifier)
}
func (f FindCharacterTextObject) GobEncode() ([]byte, error) { return encodeSessionValues(f.character) }
func (f *FindCharacterTextObject) GobDecode(data []byte) error {
	return decodeSessionValues(data, &f.character)
}
func (t TillCharacterTextObject) GobEncode() ([]byte, error) { return encodeSessionValues(t.character) }
func (t *TillCharacterTextObject) GobDecode(data []byte) error {
	return decodeSessionValues(data, &t.character)
}
func (s SearchResultTextObject) GobEncode() ([]byte, error) {
	return encodeSessionValues(s.searchPattern)
}
func (s *SearchResultTextObject) GobDecode(data []byte) error {
	return decodeSessionValues(data, &s.searchPattern)
}
func (a ArgumentTextObject) GobEncode() ([]byte, error) { return encodeSessionValues(a.modifier) }
func (a *ArgumentTextObject) GobDecode(data []byte) error {
	return decodeSessionValues(data, &a.modifier)
}
func (b BlockTextObject) GobEncode() ([]byte, error)    { return encodeSessionValues(b.modifier) }
func (b *BlockTextObject) GobDecode(data []byte) error  { return decodeSessionValues(data, &b.modifier) }
func (f FunctionTextObject) GobEncode() ([]byte, error) { return encodeSessionValues(f.modifier) }
func (f *FunctionTextObject) GobDecode(data []byte) error {
	return decodeSessionValues(data, &f.modifier)
}
