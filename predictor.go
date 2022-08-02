package nsfw

import (
	"fmt"
	tf "github.com/galeone/tensorflow/tensorflow/go"
	"github.com/galeone/tensorflow/tensorflow/go/op"
	tg "github.com/galeone/tfgo"
	"github.com/galeone/tfgo/image"
)

const (
	ImageDimensions = 224
)

type Predictor struct {
	scope *op.Scope
	model *tg.Model
}

type Prediction struct {
	Drawings float32
	Hentai   float32
	Neutral  float32
	Porn     float32
	Sexy     float32
}

func NewPredictor(model *tg.Model) *Predictor {
	return &Predictor{
		model: model,
	}
}

func NewLatestPredictor() (*Predictor, error) {
	path, err := GetLatestModelPath()
	if err != nil {
		return nil, err
	}

	return NewPredictor(path.GetModel()), nil
}

func (p *Predictor) UseScope(s *op.Scope) {
	p.scope = s
}

func (p *Predictor) setupScope() {
	if p.scope != nil || p.scope.Err() != nil {
		p.scope = tg.NewRoot()
	}
}

func (p *Predictor) NewImage(filepath string, channels int64) *image.Image {
	p.setupScope()

	return image.Read(p.scope, filepath, channels).
		ResizeArea(image.Size{Height: ImageDimensions, Width: ImageDimensions})
}

func (p *Predictor) Predict(img *image.Image) Prediction {
	p.setupScope()

	preprocess := tg.Exec(p.scope, []tf.Output{img.Output}, nil, nil)

	results := p.model.Exec([]tf.Output{
		p.model.Op("StatefulPartitionedCall", 0),
	}, map[tf.Output]*tf.Tensor{
		p.model.Op("serving_default_input", 0): preprocess[0],
	})

	vals := results[0].Value().([][]float32)[0]
	return Prediction{
		Drawings: vals[0],
		Hentai:   vals[1],
		Neutral:  vals[2],
		Porn:     vals[3],
		Sexy:     vals[4],
	}
}

func (p Prediction) Describe() string {
	return fmt.Sprintf(
		"[Drawing: %.2f%% , Hentai: %.2f%%, Porn: %.2f%%, Sexy: %.2f%%, Neutral: %.2f%%]",
		p.Drawings*100, p.Hentai*100, p.Porn*100, p.Sexy*100, p.Neutral*100)
}
