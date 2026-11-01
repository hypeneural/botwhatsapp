package relay

import (
    "os"
    "path/filepath"
    "reflect"
    "strings"
    "testing"
    "time"
)

func TestPendingIndexPersistsAndHonorsPartialSinkAcks(t *testing.T) {
    dir := t.TempDir()
    s,err:=NewStore(dir,200)
    if err!=nil {t.Fatal(err)}
    first:=strings.Repeat("a",64)
    second:=strings.Repeat("b",64)
    if ok,err:=s.Insert(first,[]byte(`{"event":"Message"}`));err!=nil||!ok {t.Fatal(ok,err)}
    if ok,err:=s.Insert(second,[]byte(`{"event":"Message"}`));err!=nil||!ok {t.Fatal(ok,err)}
    due,err:=s.Pending(time.Now())
    if err!=nil||!reflect.DeepEqual(due,[]string{first,second}){t.Fatalf("unsorted initial due index: %v %v",due,err)}
    future:=time.Now().Add(time.Hour)
    if err:=s.Postpone(second,future);err!=nil {t.Fatal(err)}
    if _,err:=s.UpdateSink(first,true,time.Now());err!=nil {t.Fatal(err)}
    due,err=s.Pending(time.Now())
    if err!=nil||!reflect.DeepEqual(due,[]string{first}){t.Fatalf("partial sink/due index: %v %v",due,err)}
    // Reboot must reconstruct only pending receipts from the on-disk spool.
    restarted,err:=NewStore(dir,200)
    if err!=nil {t.Fatal(err)}
    due,err=restarted.Pending(time.Now())
    if err!=nil||!reflect.DeepEqual(due,[]string{first}){t.Fatalf("recovery changed pending: %v %v",due,err)}
    if _,err:=restarted.UpdateSink(first,false,time.Now());err!=nil {t.Fatal(err)}
    due,err=restarted.Pending(time.Now())
    if err!=nil||len(due)!=0{t.Fatalf("completed receipt in pending index: %v %v",due,err)}
    due,err=restarted.Pending(future.Add(time.Second))
    if err!=nil||!reflect.DeepEqual(due,[]string{second}){t.Fatalf("future retry missing: %v %v",due,err)}
    again,err:=NewStore(dir,200)
    if err!=nil {t.Fatal(err)}
    due,err=again.Pending(time.Now())
    if err!=nil||len(due)!=0{t.Fatalf("completed tombstone was scheduled: %v %v",due,err)}
    if ok,err:=again.Insert(first,[]byte(`{"event":"Message"}`));err!=nil||ok{t.Fatalf("duplicate replayed: %t %v",ok,err)}
}

func TestPendingIndexRejectsCorruptSpoolOnStartup(t *testing.T){
    dir:=t.TempDir()
    if err:=os.WriteFile(filepath.Join(dir,strings.Repeat("c",64)+".json"),[]byte("{"),0600);err!=nil{t.Fatal(err)}
    if _,err:=NewStore(dir,200);err==nil{t.Fatal("corrupt durable receipt must block restart")}
}

func TestPendingIndexDoesNotTreatCompletedTombstonesAsWork(t *testing.T){
    dir:=t.TempDir()
    s,err:=NewStore(dir,200)
    if err!=nil{t.Fatal(err)}
    for i:=0;i<30;i++{
        k:=strings.Repeat(string(rune('a'+i%6)),63)+string(rune('0'+i%10))
        if ok,err:=s.Insert(k,[]byte(`{"event":"Message"}`));err!=nil||!ok{t.Fatal(err)}
        if _,err:=s.UpdateSink(k,true,time.Now());err!=nil{t.Fatal(err)}
        if _,err:=s.UpdateSink(k,false,time.Now());err!=nil{t.Fatal(err)}
    }
    due,err:=s.Pending(time.Now())
    if err!=nil||len(due)!=0{t.Fatalf("completed-only cache must be empty: %v %v",due,err)}
    again,err:=NewStore(dir,200)
    if err!=nil{t.Fatal(err)}
    due,err=again.Pending(time.Now())
    if err!=nil||len(due)!=0{t.Fatalf("boot should not enqueue tombstones: %v %v",due,err)}
}
