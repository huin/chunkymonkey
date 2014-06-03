package gamerules

import (
	"errors"
	"nbt"
)

func makeSignAspect() (aspect IBlockAspect) {
	return &SignAspect{}
}

type signTileEntity struct {
	tileEntity
	text [4]string
}

func NewSignTileEntity() ITileEntity {
	return &signTileEntity{}
}

func (sign *signTileEntity) UnmarshalNbt(tag *nbt.Compound) (err error) {
	if err = sign.tileEntity.UnmarshalNbt(tag); err != nil {
		return
	}

	var textTags [4]*nbt.String
	var ok bool

	if textTags[0], ok = tag.Lookup("Text1").(*nbt.String); !ok {
		return errors.New("tile entity sign missing or bad type Text1")
	}
	if textTags[1], ok = tag.Lookup("Text2").(*nbt.String); !ok {
		return errors.New("tile entity sign missing or bad type Text2")
	}
	if textTags[2], ok = tag.Lookup("Text3").(*nbt.String); !ok {
		return errors.New("tile entity sign missing or bad type Text3")
	}
	if textTags[3], ok = tag.Lookup("Text4").(*nbt.String); !ok {
		return errors.New("tile entity sign missing or bad type Text4")
	}

	for i, textTag := range textTags {
		sign.text[i] = textTag.Value
	}

	return nil
}

func (sign *signTileEntity) MarshalNbt(tag *nbt.Compound) (err error) {
	if err = sign.tileEntity.MarshalNbt(tag); err != nil {
		return
	}

	tag.Set("id", &nbt.String{"Sign"})
	tag.Set("Text1", &nbt.String{sign.text[0]})
	tag.Set("Text2", &nbt.String{sign.text[1]})
	tag.Set("Text3", &nbt.String{sign.text[2]})
	tag.Set("Text4", &nbt.String{sign.text[3]})

	return nil
}

type SignAspect struct {
	StandardAspect
}

func (aspect SignAspect) Name() string {
	return "Sign"
}

// TODO behaviours for signs.
