package whispermodel

import "fmt"

const sourceBase = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/"

type Model struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Type        string `json:"type"`
	SizeBytes   int64  `json:"sizeBytes"`
	Quality     int    `json:"quality"`
	Speed       string `json:"speed"`
	Memory      string `json:"memory"`
	Description string `json:"description"`
}

var Catalog = []Model{
	{ID: "tiny", Name: "Tiny", Filename: "ggml-tiny.bin", Type: "Full", SizeBytes: 75 << 20, Quality: 2, Speed: "Очень высокая", Memory: "Низкое", Description: "Самая компактная модель для быстрых черновых субтитров."},
	{ID: "tiny-q5_1", Name: "Tiny Q5", Filename: "ggml-tiny-q5_1.bin", Type: "Q5", SizeBytes: 31 << 20, Quality: 2, Speed: "Очень высокая", Memory: "Очень низкое", Description: "Квантованная Tiny с минимальным расходом диска и памяти."},
	{ID: "base", Name: "Base", Filename: "ggml-base.bin", Type: "Full", SizeBytes: 142 << 20, Quality: 2, Speed: "Высокая", Memory: "Низкое", Description: "Небольшой шаг к лучшему распознаванию без высокой нагрузки."},
	{ID: "base-q5_1", Name: "Base Q5", Filename: "ggml-base-q5_1.bin", Type: "Q5", SizeBytes: 57 << 20, Quality: 2, Speed: "Высокая", Memory: "Низкое", Description: "Компактный вариант Base для систем с ограниченными ресурсами."},
	{ID: "small", Name: "Small", Filename: "ggml-small.bin", Type: "Full", SizeBytes: 466 << 20, Quality: 3, Speed: "Средняя", Memory: "Среднее", Description: "Практичный баланс качества и скорости для локальной работы."},
	{ID: "small-q5_1", Name: "Small Q5", Filename: "ggml-small-q5_1.bin", Type: "Q5", SizeBytes: 181 << 20, Quality: 3, Speed: "Выше средней", Memory: "Ниже среднего", Description: "Уменьшенная Small с небольшой потенциальной потерей качества."},
	{ID: "medium", Name: "Medium", Filename: "ggml-medium.bin", Type: "Full", SizeBytes: 1530 << 20, Quality: 4, Speed: "Низкая", Memory: "Высокое", Description: "Высокое качество для сложной речи, требует заметно больше ресурсов."},
	{ID: "medium-q5_0", Name: "Medium Q5", Filename: "ggml-medium-q5_0.bin", Type: "Q5", SizeBytes: 539 << 20, Quality: 4, Speed: "Средняя", Memory: "Среднее", Description: "Более доступная по памяти версия Medium."},
	{ID: "large-v3-turbo", Name: "Large v3 Turbo", Filename: "ggml-large-v3-turbo.bin", Type: "Full", SizeBytes: 1550 << 20, Quality: 4, Speed: "Средняя", Memory: "Высокое", Description: "Быстрая модель высокого качества для повседневной транскрибации."},
	{ID: "large-v3-turbo-q5_0", Name: "Large v3 Turbo Q5", Filename: "ggml-large-v3-turbo-q5_0.bin", Type: "Q5", SizeBytes: 547 << 20, Quality: 4, Speed: "Выше средней", Memory: "Среднее", Description: "Квантованная Turbo: заметно меньше памяти и диска."},
	{ID: "large-v3", Name: "Large v3", Filename: "ggml-large-v3.bin", Type: "Full", SizeBytes: 2952 << 20, Quality: 5, Speed: "Очень низкая", Memory: "Очень высокое", Description: "Максимальное качество, когда скорость и расход памяти вторичны."},
	{ID: "large-v3-q5_0", Name: "Large v3 Q5", Filename: "ggml-large-v3-q5_0.bin", Type: "Q5", SizeBytes: 1030 << 20, Quality: 4, Speed: "Низкая", Memory: "Высокое", Description: "Large v3 с меньшими требованиями и небольшой потенциальной потерей качества."},
}

func Find(id string) (Model, error) {
	for _, model := range Catalog {
		if model.ID == id {
			return model, nil
		}
	}
	return Model{}, fmt.Errorf("unknown Whisper model")
}

func downloadURL(model Model) string { return sourceBase + model.Filename }
