package dispatcher

type TrackState int

const (
	StateIdle TrackState = iota
	StateWaitingTrackLink
	StateWaitingTrackTags
)

type TrackDialog struct {
	State TrackState
	Link  string
}

// Stores states of dialog for /track dialog
type StateStorage struct {
	dialogs map[int64]TrackDialog
}

func NewStateStorage() *StateStorage {
	return &StateStorage{
		dialogs: make(map[int64]TrackDialog),
	}
}

func (s *StateStorage) Get(chatID int64) TrackDialog {
	dialog, exists := s.dialogs[chatID]
	if !exists {
		return TrackDialog{State: StateIdle}
	}

	return dialog
}

func (s *StateStorage) Set(chatID int64, dialog TrackDialog) {
	s.dialogs[chatID] = dialog
}

func (s *StateStorage) Reset(chatID int64) {
	delete(s.dialogs, chatID)
}