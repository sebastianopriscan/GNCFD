package endpoints

import (
	"errors"

	"github.com/sebastianopriscan/GNCFD/core/guid"
	"github.com/sebastianopriscan/GNCFD/core/impl/vivaldi"
	"github.com/sebastianopriscan/GNCFD/gossip/rpc/grpc/vivaldi/pb_go"
)

const core_code string = "Vivaldi"

func asPointsFloat(updates *vivaldi.VivaldiMetadata[float64]) []*pb_go.NodeState {
	retVal := make([]*pb_go.NodeState, 0)

	for k, v := range updates.Data {
		coordinates := v.Coords
		coordReal := &pb_go.CoordStream{Coords: coordinates}
		point := &pb_go.Point{Dimension: int64(len(coordinates)), CoordReal: coordReal}
		retVal = append(retVal, &pb_go.NodeState{Guid: k.String(), Coords: point, Failed: v.IsFailed})
	}

	return retVal
}

func asPointsCmplx(updates *vivaldi.VivaldiMetadata[complex128]) []*pb_go.NodeState {
	retVal := make([]*pb_go.NodeState, 0)

	for k, v := range updates.Data {
		coordinates := v.Coords
		re_coords := make([]float64, 0)
		im_coords := make([]float64, 0)

		for _, coord := range coordinates {
			re_coords = append(re_coords, real(coord))
			im_coords = append(im_coords, imag(coord))
		}

		coordReal := &pb_go.CoordStream{Coords: re_coords}
		coordIm := &pb_go.CoordStream{Coords: im_coords}
		point := &pb_go.Point{Dimension: int64(len(coordinates)), CoordReal: coordReal, CoordIm: coordIm}
		retVal = append(retVal, &pb_go.NodeState{Guid: k.String(), Coords: point, Failed: v.IsFailed})
	}

	return retVal
}

func asNodeDataReal(array []*pb_go.NodeState) (map[guid.Guid]vivaldi.VivaldiMetaCoor[float64], error) {

	retVal := make(map[guid.Guid]vivaldi.VivaldiMetaCoor[float64])

	for i := 0; i < len(array); i++ {
		guid, err := guid.Deserialize([]byte(array[i].Guid))
		if err != nil {
			return nil, errors.New("error: wrong guid format")
		}

		nodeData := vivaldi.VivaldiMetaCoor[float64]{}
		nodeData.IsFailed = array[i].Failed
		nodeData.Coords = array[i].Coords.CoordReal.Coords

		retVal[guid] = nodeData
	}

	return retVal, nil
}

func asNodeDataCmplx(array []*pb_go.NodeState) (map[guid.Guid]vivaldi.VivaldiMetaCoor[complex128], error) {

	retVal := make(map[guid.Guid]vivaldi.VivaldiMetaCoor[complex128])

	for i := 0; i < len(array); i++ {
		guid, err := guid.Deserialize([]byte(array[i].Guid))
		if err != nil {
			return nil, errors.New("error deserializing a node's guid")
		}

		nodeData := vivaldi.VivaldiMetaCoor[complex128]{}
		nodeData.IsFailed = array[i].Failed

		cmplxCoords := make([]complex128, 0)

		for j := int64(0); j < array[i].Coords.Dimension; j++ {

			re := array[i].Coords.CoordReal.Coords[j]
			im := array[i].Coords.CoordIm.Coords[j]

			cmplxCoords = append(cmplxCoords, complex(re, im))
		}

		nodeData.Coords = cmplxCoords

		retVal[guid] = nodeData
	}

	return retVal, nil
}
