package matroska

import (
	"time"
	"strconv"
)

// File represents a Matroska file.
// See detailed specification https://matroska.org/technical/specs/index.html
type File struct {
	EBML    *EBML      `ebml:"1A45DFA3"`
	Segment []*Segment `ebml:"18538067"`
}

// The EBML top level element contains a description of the file type, such as EBML
// version, file type name, file type version etc.
type EBML struct {
	Version            int    `ebml:"4286"`
	ReadVersion        int    `ebml:"42F7"`
	MaxIDLength        int    `ebml:"42F2"`
	MaxSizeLength      int    `ebml:"42F3"`
	DocType            string `ebml:"4282"`
	DocTypeVersion     int    `ebml:"4287"`
	DocTypeReadVersion int    `ebml:"4285"`
}

// NewEBML creates EBML top level element with default values
func NewEBML() *EBML {
	return &EBML{1, 1, 4, 8, "matroska", 1, 1}
}

// A Segment contains multimedia data, as well as any header data necessary for replay.
type Segment struct {
	Info        []*SegmentInfo `ebml:"1549A966"`
	SeekHead    []*SeekHead    `ebml:"114D9B74,omitempty"`
	Tracks      []*Track  `ebml:"1654AE6B>AE,omitempty"`
	Cluster     []*Cluster `ebml:"1F43B675,omitempty"`
	Cues        []*CuePoint  `ebml:"1C53BB6B>BB,omitempty"`
	Attachments []*Attachment  `ebml:"1941A469>61A7"`
	Chapters    []*Chapter  `ebml:"1043A770>45B9"`
	Tags        []*Tag  `ebml:"1254C367>7373"`
}

// SegmentInfo contains general information about a segment, like an UID, a title etc.
// This information is not really required for playback.
type SegmentInfo struct {
	UID           []byte     `ebml:"73A4,omitempty"`
	Filename      string     `ebml:"7384,omitempty"`
	PrevUID       []byte     `ebml:"3CB923,omitempty"`
	PrevFilename  string     `ebml:"3C83AB,omitempty"`
	NextUID       []byte     `ebml:"3EB923,omitempty"`
	NextFilename  string     `ebml:"3E83BB,omitempty"`
	TimecodeScale int64      `ebml:"2AD7B1"`
	Duration      float64    `ebml:"4489,omitempty"`
	DateUTC       *time.Time `ebml:"4461,omitempty"`
	Title         string     `ebml:"7BA9,omitempty"`
	MuxingApp     string     `ebml:"4D80"`
	WritingApp    string     `ebml:"5741"`
	SegmentFamily []byte `ebml:"4444"`
	Translate     []*ChapterTranslate `ebml:"6924"`
}

type ChapterTranslate struct {
	EditionUID int64 `ebml:"69FC,omitempty"`
	Codec      int64 `ebml:"69BF"`
	ID         []byte `ebml:"69A5"`
}

func NewSegmentInfo() *SegmentInfo {
	return &SegmentInfo{TimecodeScale: 1000000}
}

// A SeekHead is an index of elements that are children of Segment.
// It can point to other SeekHeads, but not to itself.
// If all non-Cluster precede all Clusters, a SeekHead is not really necessary.
// Otherwise, a missing SeekHead leads to long file loading times or the inability
// to access certain data.
type SeekHead struct {
	Seek []*Seek `ebml:"4DBB"`
}

// A Seek element contains an ID and the position within the Segment at which
// an element with this ID can be found.
type Seek struct {
	ID       []byte `ebml:"53AB"`
	Position int64  `ebml:"53AC"`
}

// A Track element describes one track of the Segment.
// A file containing only chapters and attachments does not have a Track element,
// thus it’s not mandatory.
type Track struct {
	TrackNumber                 int    `ebml:"D7"`
	TrackUID                    int64  `ebml:"73C5"`
	TrackType                   int    `ebml:"83"`
	FlagEnabled                 int    `ebml:"B9"`
	FlagDefault                 int    `ebml:"88"`
	FlagForced                  int    `ebml:"55AA"`
	FlagLacing                  int    `ebml:"9C"`
	MinCache                    int    `ebml:"6DE7"`
	MaxCache                    int    `ebml:"6DF8,omitempty"`
	DefaultDuration             int64  `ebml:"23E383,omitempty"`
	DefaultDecodedFieldDuration int64  `ebml:"234E7A,omitempty"`
	MaxBlockAdditionID          int64  `ebml:"55EE"`
	Name                        string `ebml:"536E,omitempty"`
	Language                    string `ebml:"22B59C,omitempty"`
	CodecID                     string `ebml:"86"`
	CodecPrivate                []byte `ebml:"63A2,omitempty"`
	CodecName                   string `ebml:"258688,omitempty"`
	AttachmentLink              int64  `ebml:"7446,omitempty"`
	CodecDecodeAll              int64 `ebml:"AA"`
	TrackOverlay                int64 `ebml:"6FAB,omitempty"`
	CodecDelay                  int64 `ebml:"56AA,omitempty"`
	SeekPreRoll                 int64 `ebml:"56BB"`
	Translate                   []*TrackTranslate `ebml:"6624,omitempty"`
	Video                       *VideoTrack `ebml:"E0,omitempty"`
	Audio                       *AudioTrack `ebml:"E1,omitempty"`
	TrackOperation              *TrackOperation `ebml:"E2,omitempty"`
	ContentEncodings            []*ContentEncoding `ebml:"6D80>6240"`
}

// TrackTranslate describes a track identification for the given Chapter Codec.
type TrackTranslate struct {
	EditionUID int64 `ebml:"66FC,omitempty"`
	Codec      int64 `ebml:"66BF"`
	ID         []byte `ebml:"66A5"`
}

// Track types
const (
	TrackTypeVideo = 0x01
	TrackTypeAudio = 0x02
	TrackTypeComplex = 0x03
	TrackTypeLogo = 0x10
	TrackTypeSubtitle = 0x11
	TrackTypeButton = 0x12
	TrackTypeControl = 0x20
)

// A Cluster contains video, audio and subtitle data.
// Note that a Matroska file could contain chapter data or attachments,
// but no multimedia data, so Cluster is not a mandatory element.
type Cluster struct {
	Timecode     int64 `ebml:"E7"`
	SilentTracks []int `ebml:"5854>58D7,omitempty"`
	Position     int64 `ebml:"A7,omitempty"`
	PrevSize     int64 `ebml:"AB,omitempty"`
	SimpleBlock  []*Block `ebml:"A3,omitempty"`
	BlockGroup   []*BlockGroup `ebml:"A0,omitempty"`
}

type Block struct {
	Data []byte
}

func (r *Block) UnmarshalEBML(dec *ebml.Decoder) (err error) {
	r.Data, err = dec.ReadBytes()
	return
}

func (r *Block) String() string {
	return "Block{" + strconv.Itoa(len(r.Data)) + " bytes}"
}

type BlockGroup struct {
	Block             *Block `ebml:"A1"`
	Additions         []*BlockAddition `ebml:"75A1>A6,omitempty"`
	Duration          int64 `ebml:"9B,omitempty"`
	ReferencePriority int64  `ebml:"FA"`
	ReferenceBlock    int64  `ebml:"FB,omitempty"`
	CodecState        []byte `ebml:"A4,omitempty"`
	DiscardPadding    int64  `ebml:"75A2,omitempty"`
	Slices            []int64  `ebml:"8E>E8>CC,omitempty"`
}

type BlockAddition struct {
	ID         int64  `ebml:"EE"`
	Additional []byte  `ebml:"A5"`
}

// VideoTrack contains information that is specific for video tracks.
type VideoTrack struct {
	FlagInterlaced  int `ebml:"9A"`
	FieldOrder      int `ebml:"9D"`
	StereoMode      int `ebml:"53B8,omitempty"`
	AlphaMode       int `ebml:"53C0,omitempty"`
	PixelWidth      int `ebml:"B0"`
	PixelHeight     int `ebml:"BA"`
	PixelCropBottom int `ebml:"54AA,omitempty"`
	PixelCropTop    int `ebml:"54BB,omitempty"`
	PixelCropLeft   int `ebml:"54CC,omitempty"`
	PixelCropRight  int `ebml:"54DD,omitempty"`
	DisplayWidth    int `ebml:"54B0,omitempty"`
	DisplayHeight   int `ebml:"54BA,omitempty"`
	DisplayUnit     int `ebml:"54B2,omitempty"`
	AspectRatioType int `ebml:"54B3,omitempty"`
	ColourSpace     []byte `ebml:"2EB524,omitempty"`
	Colour          []*Colour `ebml:"55B0,omitempty"`
}

// Colour describes the colour format settings.
type Colour struct {
	MatrixCoefficients      int `ebml:"55B1,omitempty"`
	BitsPerChannel          int `ebml:"55B2,omitempty"`
	ChromaSubsamplingHorz   int `ebml:"55B3,omitempty"`
	ChromaSubsamplingVert   int `ebml:"55B4,omitempty"`
	CbSubsamplingHorz       int `ebml:"55B5,omitempty"`
	CbSubsamplingVert       int `ebml:"55B6,omitempty"`
	ChromaSitingHorz        int `ebml:"55B7,omitempty"`
	ChromaSitingVert        int `ebml:"55B8,omitempty"`
	Range                   int `ebml:"55B9,omitempty"`
	TransferCharacteristics int `ebml:"55BA,omitempty"`
	Primaries               int `ebml:"55BB,omitempty"`
	MaxCLL                  int `ebml:"55BC,omitempty"`
	MaxFALL                 int `ebml:"55BD,omitempty"`
	MasteringMetadata       []*MasteringMetadata `ebml:"55D0"`
}

// MasteringMetadata represents SMPTE 2086 mastering data.
type MasteringMetadata struct {
	PrimaryRChromaX   float64 `ebml:"55D1,omitempty"`
	PrimaryRChromaY   float64 `ebml:"55D2,omitempty"`
	PrimaryGChromaX   float64 `ebml:"55D3,omitempty"`
	PrimaryGChromaY   float64 `ebml:"55D4,omitempty"`
	PrimaryBChromaX   float64 `ebml:"55D5,omitempty"`
	PrimaryBChromaY   float64 `ebml:"55D6,omitempty"`
	WhitePointChromaX float64 `ebml:"55D7,omitempty"`
	WhitePointChromaY float64 `ebml:"55D8,omitempty"`
	LuminanceMax      float64 `ebml:"55D9,omitempty"`
	LuminanceMin      float64 `ebml:"55DA,omitempty"`
}

// AudioTrack contains information that is specific for audio tracks.
type AudioTrack struct {
	SamplingFreq       float64 `ebml:"B5"`
	OutputSamplingFreq float64 `ebml:"78B5,omitempty"`
	Channels           int `ebml:"9F"`
	BitDepth           int `ebml:"6264,omitempty"`
}

// TrackOperation describes an operation that needs to be applied on tracks
// to create this virtual track.
type TrackOperation struct {
	CombinePlanes []*TrackPlane `ebml:"E3>E4,omitempty"`
	JoinBlocks    []int64 `ebml:"E9>ED"`
}

// TrackPlane contains a video plane track that need to be combined to create this track.
type TrackPlane struct {
	UID  int64 `ebml:"E5"`
	Type int `ebml:"E6"`
}

// ContentEncodings contains settings for several content encoding mechanisms
// like compression or encryption.
type ContentEncoding struct {
	Order       int `ebml:"5031"`
	Scope       int `ebml:"5032"`
	Type        int `ebml:"5033"`
	Compression *Compression `ebml:"5034,omitempty"`
	Encryption  *Encryption `ebml:"5035,omitempty"`
}

// Compression describes the compression used.
type Compression struct {
	Algo     int `ebml:"4254"`
	Settings []byte `ebml:"4255,omitempty"`
}

// Encryption describes the encryption used.
type Encryption struct {
	EncAlgo     int `ebml:"47E1,omitempty"`
	EncKeyID    []byte `ebml:"47E2,omitempty"`
	Signature   []byte `ebml:"47E3,omitempty"`
	SigKeyID    []byte `ebml:"47E4,omitempty"`
	SigAlgo     int `ebml:"47E5,omitempty"`
	SigHashAlgo int `ebml:"47E6,omitempty"`
}

// CuePoint contains all information relative to a seek point in the Segment.
type CuePoint struct {
	Time           int64 `ebml:"B3"`
	TrackPositions []*CueTrackPosition `ebml:"B7"`
}

// CueTrackPosition contains positions for different tracks corresponding to the timestamp.
type CueTrackPosition struct {
	Track            int64 `ebml:"F7"`
	ClusterPosition  int64 `ebml:"F1"`
	RelativePosition int64 `ebml:"F0,omitempty"`
	Duration         int64 `ebml:"B2,omitempty"`
	BlockNumber      int64 `ebml:"5378,omitempty"`
	CodecState       int64 `ebml:"EA,omitempty"`
	References       []int64 `ebml:"DB>96,omitempty"`
}

// Attachment describes attached files.
type Attachment struct {
	UID         int64 `ebml:"46AE"`
	Description string `ebml:"467E,omitempty"`
	Name        string `ebml:"466E"`
	MimeType    string `ebml:"4660"`
	Data        []byte `ebml:"465C"`
}

// Chapter contains all information about a Segment edition.
type Chapter struct {
	EditionUID  int64 `ebml:"45BC,omitempty"`
	FlagHidden  int `ebml:"45BD"`
	FlagDefault int `ebml:"45DB"`
	FlagOrdered int `ebml:"45DD,omitempty"`
	Atoms       []*ChapterAtom `ebml:"B6"`
}

// ChapterAtom contains the atom information to use as the chapter atom (apply to all tracks).
type ChapterAtom struct {
	ChapterUID        int64 `ebml:"73C4"`
	StringUID         string `ebml:"5654,omitempty"`
	TimeStart         int64 `ebml:"91"`
	TimeEnd           int64 `ebml:"92,omitempty"`
	FlagHidden        int `ebml:"98"`
	FlagEnabled       int `ebml:"4598"`
	SegmentUID        int64 `ebml:"6E67,omitempty"`
	SegmentEditionUID int64 `ebml:"6EBC,omitempty"`
	PhysicalEquiv     int `ebml:"63C3,omitempty"`
	Tracks            []int64 `ebml:"8F>89,omitempty"`
	Displays           []*ChapterDisplay `ebml:"80,omitempty"`
	Processes           []*ChapterProcess `ebml:"6944,omitempty"`
}

// ChapterDisplay contains all possible strings to use for the chapter display.
type ChapterDisplay struct {
	String   string `ebml:"85"`
	Language string `ebml:"437C"`
	Country  string  `ebml:"437E,omitempty"`
}

// ChapterProcess describes the atom processing commands.
type ChapterProcess struct {
	CodecID int `ebml:"6955"`
	Private []byte `ebml:"450D,omitempty"`
	Command []*ChapterCommand `ebml:"6911,omitempty"`
}

// ChapterProcess contains all the commands associated to the atom.
type ChapterCommand struct {
	Time int64 `ebml:"6922"`
	Data []byte `ebml:"6933"`
}

// Tag contains meta data for tracks and/or chapters.
type Tag struct {
	Targets   []*TagTarget `ebml:"63C0"`
	Tags []*SimpleTag  `ebml:"67C8"`
}

// TagTarget contains all UIDs where the specified meta data apply.
type TagTarget struct {
	TypeValue     int `ebml:"68CA,omitempty"`
	Type          string `ebml:"63CA,omitempty"`
	TrackUID      []int64 `ebml:"63C5,omitempty"`
	EditionUID    []int64 `ebml:"63C9,omitempty"`
	ChapterUID    []int64 `ebml:"63C4,omitempty"`
	AttachmentUID []int64 `ebml:"63C6,omitempty"`
}

// SimpleTag contains general information about the target.
type SimpleTag struct {
	Tags []*SimpleTag  `ebml:"67C8"`
	Name     string `ebml:"45A3"`
	Language string `ebml:"447A"`
	Default  int `ebml:"4484"`
	String   string `ebml:"4487,omitempty"`
	Binary   []byte `ebml:"4485,omitempty"`
}
