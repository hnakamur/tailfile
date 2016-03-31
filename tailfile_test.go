package tailfile

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
)

func ExampleTailCreateRenameRecreate() {
	dir, err := ioutil.TempDir("", "tailfile-example1")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "example.log")
	renamedPath := filepath.Join(dir, "example.log.old")

	done := make(chan struct{})

	go func() {
		defer func() {
			done <- struct{}{}
		}()

		interval := time.Duration(9) * time.Millisecond
		file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal(err)
		}

		i := 0
		for ; i < 50; i++ {
			_, err := file.WriteString(fmt.Sprintf("line%d\n", i))
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(interval)
		}

		err = os.Rename(targetPath, renamedPath)
		if err != nil {
			log.Fatal(err)
		}
		for ; i < 100; i++ {
			_, err := file.WriteString(fmt.Sprintf("line%d\n", i))
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(interval)
		}
		err = file.Close()
		if err != nil {
			log.Fatal(err)
		}

		file, err = os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal()
		}
		for ; i < 150; i++ {
			_, err := file.WriteString(fmt.Sprintf("line%d\n", i))
			if err != nil {
				log.Fatal(err)
			}
			time.Sleep(interval)
		}
		err = file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	t, err := NewTailFile(targetPath, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer t.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go t.ReadLoop(ctx)
loop:
	for {
		select {
		case line := <-t.Lines:
			fmt.Printf("line=%s", line)
		case err := <-t.Errors:
			fmt.Printf("error from tail. err=%s\n", err)
			break loop
		case <-done:
			fmt.Println("got done")
			cancel()
			break loop
		default:
			// do nothing
		}
	}

	// Output:
	//line=line0
	//line=line1
	//line=line2
	//line=line3
	//line=line4
	//line=line5
	//line=line6
	//line=line7
	//line=line8
	//line=line9
	//line=line10
	//line=line11
	//line=line12
	//line=line13
	//line=line14
	//line=line15
	//line=line16
	//line=line17
	//line=line18
	//line=line19
	//line=line20
	//line=line21
	//line=line22
	//line=line23
	//line=line24
	//line=line25
	//line=line26
	//line=line27
	//line=line28
	//line=line29
	//line=line30
	//line=line31
	//line=line32
	//line=line33
	//line=line34
	//line=line35
	//line=line36
	//line=line37
	//line=line38
	//line=line39
	//line=line40
	//line=line41
	//line=line42
	//line=line43
	//line=line44
	//line=line45
	//line=line46
	//line=line47
	//line=line48
	//line=line49
	//line=line50
	//line=line51
	//line=line52
	//line=line53
	//line=line54
	//line=line55
	//line=line56
	//line=line57
	//line=line58
	//line=line59
	//line=line60
	//line=line61
	//line=line62
	//line=line63
	//line=line64
	//line=line65
	//line=line66
	//line=line67
	//line=line68
	//line=line69
	//line=line70
	//line=line71
	//line=line72
	//line=line73
	//line=line74
	//line=line75
	//line=line76
	//line=line77
	//line=line78
	//line=line79
	//line=line80
	//line=line81
	//line=line82
	//line=line83
	//line=line84
	//line=line85
	//line=line86
	//line=line87
	//line=line88
	//line=line89
	//line=line90
	//line=line91
	//line=line92
	//line=line93
	//line=line94
	//line=line95
	//line=line96
	//line=line97
	//line=line98
	//line=line99
	//line=line100
	//line=line101
	//line=line102
	//line=line103
	//line=line104
	//line=line105
	//line=line106
	//line=line107
	//line=line108
	//line=line109
	//line=line110
	//line=line111
	//line=line112
	//line=line113
	//line=line114
	//line=line115
	//line=line116
	//line=line117
	//line=line118
	//line=line119
	//line=line120
	//line=line121
	//line=line122
	//line=line123
	//line=line124
	//line=line125
	//line=line126
	//line=line127
	//line=line128
	//line=line129
	//line=line130
	//line=line131
	//line=line132
	//line=line133
	//line=line134
	//line=line135
	//line=line136
	//line=line137
	//line=line138
	//line=line139
	//line=line140
	//line=line141
	//line=line142
	//line=line143
	//line=line144
	//line=line145
	//line=line146
	//line=line147
	//line=line148
	//line=line149
	//got done
}
