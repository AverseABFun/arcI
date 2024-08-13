package arci

import "slices"

type Node struct {
	Output        float64
	CalculatedOut bool
	ActivThresh   float64
	Weights       map[*Node]*float64
	SecondWeights map[*Node]*float64
	Children      []*Node
	InpActivFunc  func(float64) float64
	OutActivFunc  func(float64) float64
	GlobalFWeight float64
	GlobalSWeight float64
}

func (n *Node) AddChild(node *Node, weight float64) {
	n.Children = append(n.Children, node)
	node.Weights[n] = &weight
}

func (n *Node) SetOutput(out float64) {
	n.Output = out
	n.CalculatedOut = true
}

type number interface {
	int | uint | float32 | float64
}

func indexOfMax[T number](nums []T) int {
	if len(nums) == 0 {
		return -1 // return -1 if the slice is empty
	}

	maxIndex := 0
	for i := 1; i < len(nums); i++ {
		if nums[i] > nums[maxIndex] {
			maxIndex = i
		}
	}
	return maxIndex
}

func (n *Node) Train(learningRate float64, granularity uint, getReward func() float64) {
	var dimensions = []*float64{&n.ActivThresh, &n.GlobalFWeight, &n.GlobalSWeight}
	for _, val := range n.Weights {
		dimensions = append(dimensions, val)
	}
	for _, val := range n.SecondWeights {
		dimensions = append(dimensions, val)
	}
	for _, dimension := range dimensions {
		var orig_dimension = *dimension
		var adjustment float64 = 0
		var rewards = map[float64]float64{}
		var keys = []float64{}
		var values = []float64{}
		for adjustment <= learningRate {
			adjustment += learningRate / float64(granularity)
			(*dimension) = orig_dimension + adjustment
			var reward = getReward()
			rewards[adjustment] = reward
			keys = append(keys, adjustment)
			values = append(values, reward)
		}
		adjustment = 0
		for (-adjustment) <= learningRate {
			adjustment -= learningRate / float64(granularity)
			(*dimension) = orig_dimension + adjustment
			var reward = getReward()
			rewards[adjustment] = reward
			keys = append(keys, adjustment)
			values = append(values, reward)
		}
		var max = indexOfMax(values)
		(*dimension) = orig_dimension + keys[max]
	}
}

func (n *Node) CalculateOutput() {
	var inp = n.aggregateInput()
	inp = n.OutActivFunc(inp*n.GlobalFWeight) * n.GlobalSWeight
	if inp < n.ActivThresh {
		inp = 0
	}
	n.SetOutput(inp)
}

func (n Node) aggregateInput() float64 {
	var out float64 = 0
	for node, weight := range n.Weights {
		if !node.CalculatedOut {
			node.CalculateOutput()
		}
		out += n.InpActivFunc(node.Output*(*weight)) * (*n.SecondWeights[node])
	}
	return out
}

type Intelligence struct {
	InputNodes    []*Node
	TrainingNodes []*Node
	Circles       [][]*Node
	OutputNodes   []*Node
}

type TrainingData struct {
	Input []float64
}

func (intel Intelligence) Run(input []float64) []float64 {
	for i, inpNode := range intel.InputNodes {
		inpNode.SetOutput(input[i])
	}
	for _, node := range intel.TrainingNodes {
		node.CalculatedOut = false
	}
	var out = []float64{}
	for _, node := range intel.OutputNodes {
		node.CalculateOutput()
		out = append(out, node.Output)
	}
	return out
}

func (intel Intelligence) Train(data []TrainingData, circularDefault float64, learningRate float64, granularity uint, rewardFunc func([]float64) float64) {
	for _, data := range data {
		if len(data.Input) != len(intel.InputNodes) {
			panic("invalid training data input length")
		}
		for i, inpNode := range intel.InputNodes {
			inpNode.SetOutput(data.Input[i])
		}
		for _, node := range intel.TrainingNodes {
			node.CalculatedOut = false
		}
		for _, circle := range intel.Circles {
			for _, node := range circle {
				for n := range node.Weights {
					if slices.Contains(circle, n) {
						n.SetOutput(circularDefault)
					}
				}
				node.Train(learningRate, granularity, func() float64 {
					var output = intel.Run(data.Input)
					return rewardFunc(output)
				})
			}
		}
	}
}

func CreateIntelligence(numInput uint, numCircles uint, circleSize uint, numOutput uint, inpActivationFunc func(float64) float64, outActivationFunc func(float64) float64, defaultWeight float64, defaultThresh float64) Intelligence {
	var outputIntel = Intelligence{}
	for i := uint(0); i < numOutput; i++ {
		outputIntel.OutputNodes = append(outputIntel.OutputNodes, &Node{InpActivFunc: inpActivationFunc, OutActivFunc: outActivationFunc, ActivThresh: defaultThresh})
		outputIntel.TrainingNodes = append(outputIntel.TrainingNodes, outputIntel.OutputNodes[len(outputIntel.OutputNodes)-1])
	}
	for i := uint(0); i < numCircles; i++ {
		var newCircle = []*Node{}
		for f := uint(0); f < circleSize; f++ {
			newCircle = append(newCircle, &Node{InpActivFunc: inpActivationFunc, OutActivFunc: outActivationFunc, ActivThresh: defaultThresh})
			outputIntel.TrainingNodes = append(outputIntel.TrainingNodes, newCircle[len(newCircle)-1])
		}
		for f := uint(0); f < circleSize; f++ {
			if f < circleSize-1 {
				newCircle[f].AddChild(newCircle[f+1], defaultWeight)
			} else {
				newCircle[f].AddChild(newCircle[0], defaultWeight)
			}
		}
		if i == circleSize-1 {
			for f := uint(0); f < circleSize; f++ {
				for o := uint(0); o < numOutput; o++ {
					newCircle[f].AddChild(outputIntel.OutputNodes[o], defaultWeight)
				}
			}
		}
		outputIntel.Circles = append(outputIntel.Circles, newCircle)
	}
	for i := uint(0); i < numCircles-1; i++ {
		for f := uint(0); f < circleSize; f++ {
			outputIntel.Circles[i][f].AddChild(outputIntel.Circles[i+1][f], defaultWeight)
		}
	}
	for i := uint(0); i < numInput; i++ {
		outputIntel.InputNodes = append(outputIntel.InputNodes, &Node{})
		for f := uint(0); f < circleSize; f++ {
			outputIntel.InputNodes[len(outputIntel.InputNodes)-1].AddChild(outputIntel.Circles[0][f], defaultWeight)
		}
	}
	return outputIntel
}
