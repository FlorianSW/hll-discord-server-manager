package marshaller

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

var (
	stringTypes = []discordgo.ApplicationCommandOptionType{
		discordgo.ApplicationCommandOptionString,
		discordgo.ApplicationCommandOptionUser,
		discordgo.ApplicationCommandOptionChannel,
		discordgo.ApplicationCommandOptionRole,
		discordgo.ApplicationCommandOptionMentionable,
	}
	intTypes = []discordgo.ApplicationCommandOptionType{
		discordgo.ApplicationCommandOptionInteger,
		discordgo.ApplicationCommandOptionNumber,
	}
)

func Unmarshal(d any, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("invalid type: %s", reflect.TypeOf(v))
	}
	ref := rv.Elem()
	if ref.Kind() != reflect.Struct {
		return fmt.Errorf("only structs are supported: %s", reflect.TypeOf(v))
	}

	if c, ok := d.([]*discordgo.ApplicationCommandInteractionDataOption); ok {
		return unmarshalApplicationCommandInteractionDataOptions(c, ref)
	} else if c, ok := d.([]discordgo.MessageComponent); ok {
		return unmarshalMessageComponents(c, ref)
	}
	return errors.New("unsupported type " + reflect.TypeOf(d).String())
}

func unmarshalApplicationCommandInteractionDataOptions(d []*discordgo.ApplicationCommandInteractionDataOption, ref reflect.Value) error {
	tv := ref.Type()
	for j := 0; j < tv.NumField(); j++ {
		f := tv.Field(j)
		if f.Type.Kind() == reflect.Struct {
			return errors.New("embedded structs are not supported")
		}
		p, err := fieldName(f)
		if err != nil {
			return err
		}
		if p == nil {
			continue
		}

		fv := ref.FieldByName(f.Name)
		if !fv.CanSet() {
			return fmt.Errorf("cannot set field %s", f.Name)
		}
		var option *discordgo.ApplicationCommandInteractionDataOption
		for _, opt := range d {
			if opt.Name == *p {
				option = opt
				break
			}
		}
		if option == nil {
			continue
		}
		switch f.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if !slices.Contains(intTypes, option.Type) {
				return fmt.Errorf("cannot unmarshal option %s of type %s into field %s of type %s", *p, option.Type, f.Name, f.Type.Kind())
			}
			iv, ok := option.Value.(int)
			if !ok {
				return fmt.Errorf("cannot convert %s to int for field %s: %w", option.Name, f.Name, err)
			}
			fv.SetInt(int64(iv))
		case reflect.String:
			if !slices.Contains(stringTypes, option.Type) {
				return fmt.Errorf("cannot unmarshal option %s of type %s into field %s of type %s", *p, option.Type, f.Name, f.Type.Kind())
			}
			s, ok := option.Value.(string)
			if !ok {
				return fmt.Errorf("cannot convert %s to string for field %s: %w", option.Name, f.Name, err)
			}
			fv.SetString(s)
		case reflect.Bool:
			if option.Type != discordgo.ApplicationCommandOptionBoolean {
				return fmt.Errorf("cannot unmarshal option %s of type %s into field %s of type %s", *p, option.Type, f.Name, f.Type.Kind())
			}
			b, ok := option.Value.(bool)
			if !ok {
				return fmt.Errorf("cannot convert %s to boolean for field %s: %w", option.Name, f.Name, err)
			}
			fv.SetBool(b)
		default:
			return fmt.Errorf("cannot unmarshal into field %s of type %s", f.Name, f.Type.Kind())
		}
	}
	return nil
}

func fieldName(f reflect.StructField) (*string, error) {
	if ps, ok := f.Tag.Lookup("discordgo"); !ok {
		return nil, nil
	} else {
		return &ps, nil
	}
}

func unmarshalMessageComponents(d []discordgo.MessageComponent, ref reflect.Value) error {
	tv := ref.Type()
	for j := 0; j < tv.NumField(); j++ {
		f := tv.Field(j)
		if f.Type.Kind() == reflect.Struct {
			return errors.New("embedded structs are not supported")
		}
		p, err := fieldName(f)
		if err != nil {
			return err
		}
		if p == nil {
			continue
		}

		fv := ref.FieldByName(f.Name)
		if !fv.CanSet() {
			return fmt.Errorf("cannot set field %s", f.Name)
		}
		option := findComponent(d, *p)
		if option == nil {
			continue
		}
		switch f.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			iv, err := strconv.Atoi(option.Value)
			if err != nil {
				return fmt.Errorf("cannot convert %s to int for field %s: %w", option.Value, f.Name, err)
			}
			fv.SetInt(int64(iv))
		case reflect.String:
			fv.SetString(option.Value)
		default:
			return fmt.Errorf("cannot unmarshal into field %s of type %s", f.Name, f.Type.Kind())
		}
	}

	for _, cmp := range d {
		if t, ok := cmp.(*discordgo.ActionsRow); ok {
			if err := unmarshalMessageComponents(t.Components, ref); err != nil {
				return err
			}
		} else if t, ok := cmp.(*discordgo.TextInput); ok {
			field := ref.FieldByNameFunc(func(s string) bool {
				return strings.ToLower(s) == strings.ReplaceAll(strings.ToLower(t.CustomID), "-", "")
			})
			if !field.IsValid() {
				continue
			}
			ov := reflect.ValueOf(t.Value)
			if ov.Kind() == field.Kind() {
				field.Set(ov)
			}
		}
	}
	return nil
}

func findComponent(cmp []discordgo.MessageComponent, id string) *discordgo.TextInput {
	for _, component := range cmp {
		if component.Type() == discordgo.ActionsRowComponent {
			r := findComponent(component.(*discordgo.ActionsRow).Components, id)
			if r != nil {
				return r
			}
		} else if component.Type() == discordgo.TextInputComponent {
			t := component.(*discordgo.TextInput)
			if t.CustomID == id {
				return t
			}
		}
	}
	return nil
}
