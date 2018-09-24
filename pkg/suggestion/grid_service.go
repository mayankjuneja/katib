package suggestion

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"

	"github.com/kubeflow/katib/pkg/api"
	"google.golang.org/grpc"
)

type GridSuggestParameters struct {
	defaultGridNum int
	gridConfig     map[string]int
	MaxParallel    int
}

type GridSuggestService struct {
}

func NewGridSuggestService() *GridSuggestService {
	return &GridSuggestService{}
}

func (s *GridSuggestService) allocInt(min int, max int, reqnum int, step int) []string {
	if step == 0 {
		step = 1
	}
	if reqnum == 0 {
		reqnum = (max - min + step) / step
	}
	ret := make([]string, reqnum)
	if reqnum == 1 {
		ret[0] = strconv.Itoa(min)
	} else {
		for i := 0; i < reqnum; i++ {
			ret[i] = strconv.Itoa(min + i * step)
		}
	}
	return ret
}

func (s *GridSuggestService) allocFloat(min float64, max float64, reqnum int, step float64) []string {
	if step == 0 {
		step = 1.0
	}
	if reqnum == 0 {
		reqnum = int(math.Floor((max - min + step) / step))
	}
	ret := make([]string, reqnum)
	if reqnum == 1 {
		ret[0] = strconv.FormatFloat(min, 'f', 4, 64)
	} else {
		for i := 0; i < reqnum; i++ {
			ret[i] = strconv.FormatFloat(min + float64(i) * step, 'f', 4, 64)
		}
	}
	return ret
}

func (s *GridSuggestService) allocCat(list []string, reqnum int) []string {
	if reqnum == 0 {
		reqnum = len(list)
	}
	ret := make([]string, reqnum)
	if reqnum == 1 {
		ret[0] = list[0]
	} else {
		for i := 0; i < reqnum; i++ {
			ret[i] = list[(((len(list) - 1) * i) / (reqnum - 1))]
			fmt.Printf("ret %v %v\n", i, ret[i])
		}
	}
	return ret
}

func (s *GridSuggestService) setP(gci int, p [][]*api.Parameter, pg [][]string, pcs []*api.ParameterConfig) {
	if gci == len(pg)-1 {
		for i := range pg[gci] {
			p[i] = append(p[i], &api.Parameter{
				Name:          pcs[gci].Name,
				ParameterType: pcs[gci].ParameterType,
				Value:         pg[gci][i],
			})

		}
		return
	} else {
		d := len(p) / len(pg[gci])
		for i := range pg[gci] {
			for j := d * i; j < d*(i+1); j++ {
				p[j] = append(p[j], &api.Parameter{
					Name:          pcs[gci].Name,
					ParameterType: pcs[gci].ParameterType,
					Value:         pg[gci][i],
				})
			}
			s.setP(gci+1, p[d*i:d*(i+1)], pg, pcs)
		}
	}
}

func (s *GridSuggestService) purseSuggestParam(suggestParam []*api.SuggestionParameter) (int, map[string]int) {
	ret := make(map[string]int)
	defaultGrid := 0
	for _, sp := range suggestParam {
		switch sp.Name {
		case "DefaultGrid":
			defaultGrid, _ = strconv.Atoi(sp.Value)
		default:
			ret[sp.Name], _ = strconv.Atoi(sp.Value)
		}
	}
	return defaultGrid, ret
}
func (s *GridSuggestService) genGrids(studyId string, pcs []*api.ParameterConfig, df int, glist map[string]int) [][]*api.Parameter {
	var pg [][]string
	var grid []string
	var holenum = 1
	gcl := make([]int, len(pcs))
	for i, pc := range pcs {
		gc, ok := glist[pc.Name]
		if !ok {
			gc = df
		}
		switch pc.ParameterType {
		case api.ParameterType_INT:
			imin, _ := strconv.Atoi(pc.Feasible.Min)
			imax, _ := strconv.Atoi(pc.Feasible.Max)
			istep, _ := strconv.Atoi(pc.Feasible.Step)
			grid = s.allocInt(imin, imax, gc, istep)
			pg = append(pg, grid)
		case api.ParameterType_DOUBLE:
			dmin, _ := strconv.ParseFloat(pc.Feasible.Min, 64)
			dmax, _ := strconv.ParseFloat(pc.Feasible.Max, 64)
			dstep, _ := strconv.ParseFloat(pc.Feasible.Step, 64)
			grid = s.allocFloat(dmin, dmax, gc, dstep)
			pg = append(pg, grid)
		case api.ParameterType_CATEGORICAL:
			if (gc == 0) {
				gc = len(pc.Feasible.List);
			}
			grid = s.allocCat(pc.Feasible.List, gc)
			pg = append(pg, grid)
		}
		holenum *= len(grid)
		gcl[i] = gc

	}
	ret := make([][]*api.Parameter, holenum)
	s.setP(0, ret, pg, pcs)
	log.Printf("Study %v : %v parameters generated", studyId, holenum)
	return ret
}

func (s *GridSuggestService) GetSuggestions(ctx context.Context, in *api.GetSuggestionsRequest) (*api.GetSuggestionsReply, error) {
	conn, err := grpc.Dial(manager, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
		return &api.GetSuggestionsReply{}, err
	}
	defer conn.Close()
	c := api.NewManagerClient(conn)
	screq := &api.GetStudyRequest{
		StudyId: in.StudyId,
	}
	scr, err := c.GetStudy(ctx, screq)
	if err != nil {
		log.Fatalf("GetStudyConf failed: %v", err)
		return &api.GetSuggestionsReply{}, err
	}
	spreq := &api.GetSuggestionParametersRequest{
		ParamId: in.ParamId,
	}
	spr, err := c.GetSuggestionParameters(ctx, spreq)
	if err != nil {
		log.Fatalf("GetParameter failed: %v", err)
		return &api.GetSuggestionsReply{}, err
	}
	df, glist := s.purseSuggestParam(spr.SuggestionParameters)
	grids := s.genGrids(in.StudyId, scr.StudyConfig.ParameterConfigs.Configs, df, glist)
	var reqnum = int(in.RequestNumber)
	if reqnum == 0 {
		reqnum = len(grids)
	}
	trials := make([]*api.Trial, reqnum)
	for i := 0; i < int(reqnum); i++ {
		trials[i] = &api.Trial{
			StudyId:      in.StudyId,
			ParameterSet: grids[i],
		}
	}
	for i, t := range trials {
		req := &api.CreateTrialRequest{
			Trial: t,
		}
		ret, err := c.CreateTrial(ctx, req)
		if err != nil {
			return &api.GetSuggestionsReply{Trials: []*api.Trial{}}, err
		}
		trials[i].TrialId = ret.TrialId
	}
	return &api.GetSuggestionsReply{Trials: trials}, nil
}
