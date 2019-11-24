package main

import (
	"fmt"
	"bufio"
	"os"
	"log"
	"strconv"
	"strings"
	"sync"
)
//-------------Skaitytojai
//Pirkejas turintias lauka kieki
type Model struct {
	Field string
	Quantity int
}
//pirkejo inventorius
type ModelBox struct {
	Inventor [10]Model
}
//pirkejai
type Buyers struct {
	Buyer [10]ModelBox
}
//--------------Rasytojai
//laptopo duomenys
type Laptop struct {
	Name string
	Memory     int
	Price      float64
}
//paduotuveje laptopai
type Shop struct {
	LaptopBox [10]Laptop
}
//parduotuves
type Shops struct {
	Shop [10]Shop
}
//--------------
var SyncParrel = sync.WaitGroup{}//sinchronizatoriaus grupe
//--------------KONSTANTOS
const Writer_Count int = 5 //rasytoju kiekis
const Reader_Count int = 4 //skaitytoju kiekis
const DataFile string = "dat_3.txt" //skaitymo faials
const RezFile string = "rez.txt" //rasymo failas
//----------------
func main()  {
	//Duomenu perdavimo kanalai
	var WriterChannel = make(chan string)
	var ReaderChannel = make(chan string)
	//pabaigos perdavimo kanalas
	var EndChannel = make(chan string)
	//rasytojo pabaigos kanalas
	var ENDWriteChannel = make (chan int)
	//kanalas kuris perduoas informacija apie rasytoju pabaigima
	var WriterIsDone = make (chan bool)
	//skaitytojo pabaigos kanalas
	var ENDReaderChannel = make (chan int)
	//is duomenu proc. perdavimas i rasytojus ir skaitytojus
	var UploadDataToWriters = make(chan Shop)
	var UplaodCountToWriters = make(chan int)
	var UploadDataToReaders = make(chan ModelBox)
	var UplaodCountToReaders = make(chan int)
	//kanalai, informacija apie istrinima, idejima
	var RezReader = make (chan bool)
	var RezWriter = make (chan bool)
	SyncParrel.Add(1)//padidinama viena
	//vykdomas Duomenu perdavimo procesas		
	go ParrelCollectData(UploadDataToWriters, UplaodCountToWriters,UploadDataToReaders,UplaodCountToReaders,EndChannel, ENDWriteChannel,ENDReaderChannel,WriterIsDone)
	SyncParrel.Add(1)
	//vykdomas Valdytojo procesas	
	go ParrelControler(WriterChannel,ReaderChannel,RezReader,RezWriter,EndChannel,WriterIsDone)
	//vykodmi Rasytojai/Parduotuves procesai
	for i := 0; i < Writer_Count; i++ {
		SyncParrel.Add(1)
		go ShopsParrel(i, WriterChannel,UploadDataToWriters,UplaodCountToWriters,ENDWriteChannel,RezWriter)//rasytojai
	}
	//vykdomi Skaitytoju/Pirkeju procesai	
	for i := 0; i < Reader_Count; i++ {
		SyncParrel.Add(1)
		go BuyersParrel(i, ReaderChannel,UploadDataToReaders,UplaodCountToReaders,ENDReaderChannel,RezReader)//sakitytojai
	}
	SyncParrel.Wait()//Blokuoja pagrindine gija, kol visi baigs darba
	fmt.Println("PROGRAM DONE!")
}
//---------------------Duomenu perdavimo procesas
func ParrelCollectData(UploadDataToWriters chan<- Shop, UplaodCountToWriters chan<- int,
	UploadDataToReaders chan<- ModelBox, UplaodCountToReaders chan<- int,
	EndChannel chan<- string, ENDWriteChannel <-chan int, ENDReaderChannel <-chan int,
	WriterIsDone chan<- bool){
	defer SyncParrel.Done()//sakinys vykdomas paskytinis, sumazinoma vienu
	var shops, buyers, C_shops, C_buyers = ReadData()//duomenu skaitymas
	PrintDataToFile(shops,buyers,C_shops,C_buyers)//pradiniu duomenu rasymas
	for i := 0; i < Writer_Count; i++{//siunciami rasytojams
		UploadDataToWriters <- shops.Shop[i]//duomenys
		UplaodCountToWriters <- C_shops[i]//kiekvieno rasytojo duomenu kiekis
	}
	for i := 0; i < Reader_Count; i++{//siunciami skaitytojams
		UploadDataToReaders <- buyers.Buyer[i]
		UplaodCountToReaders <- C_buyers[i]
	}
	//Laukiami kol rasytojai bus ivykde darba
	//atsiuncia vienu kanalu rasytoju reiksmes
	for i := 0; i < Writer_Count; i++{
		<-ENDWriteChannel
	}
	WriterIsDone<-true//siunciama valdytoju kad rasytojai baige darba
	//Laukiami kol skaitytojai bus ivykde darba
	//atsiuncia vienu kanalu skaitytoju reiksmes
	for i := 0; i < Reader_Count; i++{
		<-ENDReaderChannel
   	}
	fmt.Println("SENDING END")	
	EndChannel <- "END" //siunciama valdytojui reiksme, kad pabaiga
	fmt.Println("SENT END")	
	line := fmt.Sprintf("COLLECTOR DONE!")
	fmt.Println(line)
}
//---------------------KONTEINERIS
//Valdytojo konteineris, i kuri bus dedami ir salinami duomenys
type ModelContainer struct {
	MBox [50]Model
}
//---------------------Valdytojo procesas
//Globalus kintamieji, tik valdytojas juos naudoja
var Count_Array = 0//masyvo kiekis
var Container = ModelContainer{}//Konteineris
func ParrelControler(WriterChannel <-chan string, ReaderChannel <-chan string,RezReader chan<- bool,RezWriter chan<- bool, EndChannel <-chan string,
	WriterIsDone <-chan bool){
	defer SyncParrel.Done()
	var OVER = false//uzbaigti
	var WritersDone = false//rasytoju ivykdymas
	var ri = 0//skaitymo veiksmo numerys
	var wi = 0//rasymo veiksmo numerys
	for {//begalinis ciklas
		select {
		case Field := <- WriterChannel://rasytoju kanalas any2one
			line := fmt.Sprintf("WRITE: |%10d|%15s|",wi + 1,Field) 
			fmt.Println(line)
			var add = Container.AddData(Field)//ideda lauka
			RezWriter <- add//kanalu sunciama irasymo rezultatas
			wi++
		case Field := <- ReaderChannel://skaitytoju kanalas any2one
			line := fmt.Sprintf("READ:  |%10d|%15s|",ri + 1,Field)
			fmt.Println(line) 	
			var remove = Container.RemoveData(Field)//salina lauka
			if(WritersDone){//jei rasytojai baige darba
				RezReader<-true//sius kanalu visada true, "pasalino" is kontainerio
			}else{
				RezReader <- remove//kanalu sunciama skaitymo rezultatas 
			}
			ri++
		case <- WriterIsDone://kanalu bus atsiunciama reiksme apie rasytoju pabaigima
			WritersDone = true
		case end := <- EndChannel://atsiuncia, kad valdytojas pabaigtu
			line := fmt.Sprintf("--------- %3s ----------", end) 
			fmt.Println(line)
			OVER = true
			break	
		}
		if(OVER){//kad ciklas sustotu
			break
		}
			
	}
	Container.PrintToConsole()//konteinerio info i konsole

AppendToFile(Container, Count_Array)//konteinerio info i faila
line := fmt.Sprintf("APPENDED TO FIELE!")
fmt.Println(line)
line = fmt.Sprintf("CONTROLLER DONE!")
fmt.Println(line)
}
//----------------------Rasytoju ir skaitymo procesai
//paleidziami Skaitytjai
func BuyersParrel(i int, ReaderChannel chan<- string,UploadDataToReaders <-chan ModelBox, UplaodCountToReaders <-chan int,
	ENDReaderChannel chan<- int, RezReader <-chan bool) {
	defer SyncParrel.Done()
	var buyer = <- UploadDataToReaders//duomenys
	var Count = <- UplaodCountToReaders//kiekis duomenu
	var j = 0
	for j < Count{
		ReaderChannel <- buyer.Inventor[j].Field//siuncia kanalu lauka
		var rez = <- RezReader//laukia rezultato
		if(rez == true){
			buyer.Inventor[j].Quantity--//atima kieki
			if(buyer.Inventor[j].Quantity == 0){
				j++
			}
		}
	}
	line := fmt.Sprintf("%3d READER DONE!",i)
	ENDReaderChannel <- i//siuncia kad baige darba
	fmt.Println(line)
}
//Paleidziami rasytojai
func ShopsParrel(i int, WriterChannel chan<- string,UploadDataToWriters <-chan Shop, UplaodCountToWriters <-chan int,
	ENDWriteChannel chan<- int,RezWriter <-chan bool) {
	defer SyncParrel.Done()
	var shop = <- UploadDataToWriters//duomenys
	var Count = <- UplaodCountToWriters//kiekis duomenu
	for j := 0; j < Count; j++{
		WriterChannel <-shop.LaptopBox[j].Name//siuncia kanalu lauka
		var rez = <- RezWriter//laukia rezultato
		if(rez == false){
			j--//jei neirase griztama prie to pacio lauko
		}
	}
	line := fmt.Sprintf("%3d WRITER DONE!",i)
	ENDWriteChannel <- i//siuncia kad baige darba
	fmt.Println(line)
}
//----------Ideti ir Salinti ir kiti konteinerio metodai-------------//
//----------Ideda i konteineri
func(Container *ModelContainer) AddData(tfield string) (Add bool){
	var add, push, Index = CheckSpace(tfield)//tikrina vieta
	if(add == false){//jei nera false
		Add = false
		return
	}
	Add = true
	if(push == true){//jei reika laukus pastuma del tvarkos
		for j := Count_Array - 1; j >= Index; j-- {
			Container.MBox[j+1] = Container.MBox[j]
		}
		Container.MBox[Index].Field=tfield
		Container.MBox[Index].Quantity=1
		Count_Array++
	}else{
		if(Container.MBox[Index].Field != ""){
			Container.MBox[Index].Quantity++//prideda +1 prie kekio jei yra toks laukas
		}
		if(Container.MBox[Index].Field == ""){//kita ideda nauja lauka su 1 kiekiu
			Count_Array++
			Container.MBox[Index].Field = tfield
			Container.MBox[Index].Quantity = 1
		}
	}
	return
}
func CheckSpace(tfield string) (add bool, push bool, Index int){
	push = false
	add = false
	Index = 0
	for i := 0; i < Count_Array; i++{
		Index = i
		if(Container.MBox[i].Field == tfield){//kai nereikia sumti, yra toks laukas
			add = true
			push = false
			break
		}
		if(Container.MBox[i].Field > tfield){//kai reikia pastumti		
			add = true
			push = true
			break
		}
	}
	if(add == false){//jei salyga add=false
		if(Index != 49){//tikrina ar nepaskutinis
			add = true//ides pirma
			if(Count_Array!=0){//ides bet kokiu atveju i gala
				Index++
			}
		}
	}
	return
}
//----------Spausdina konteinerio duomenys i konsole
func(Container *ModelContainer) PrintToConsole(){
	line := fmt.Sprintf("COUNT: %3d",Count_Array)
	fmt.Println(line)
	for i := 0; i < Count_Array; i++{
		line := fmt.Sprintf("|%3d|%16s|%15d|",i+1,Container.MBox[i].Field,Container.MBox[i].Quantity)
		fmt.Println(line)
	}
}
//--------Salina is Konteinerio
func(Container *ModelContainer) RemoveData(tfield string) (Remove bool){
	var remove, Index = CheckField(tfield)//tikrina ar yra toks laukas
	if(remove){
		if(Container.MBox[Index].Quantity <= 1){//tikrina ar kiekis 1 ar maziau, jei taip, tai perkelia laukus del tvarkos
			for i := Index; i < Count_Array; i++ {
				Container.MBox[i].Field = Container.MBox[i+1].Field
				Container.MBox[i].Quantity = Container.MBox[i+1].Quantity
			}
			Count_Array--
		}else{//kitu atvejus sumazins kieki vienu
			Container.MBox[Index].Quantity--
		}
		Remove = true
	}else{//jei nera tokio lauko
		Remove = false
	}
	return
}
func CheckField(tfield string) (remove bool, Index int){
	Index = 0
	remove = false
	for i := 0; i < Count_Array; i++ {
		if(Container.MBox[i].Field == tfield){//tikrina ar laukas egzistuoja konteinery
			Index = i
			remove = true
			break
		}
	}
	return
}
//--------------Duomenu skaitymas ir rasymas--------------
//----------Duomenu skaitymas is failo
//Grazina
//Parduotuviu, Pirkeju duomenis
//kiekius apie duomenius
func ReadData() (shops Shops, buyers Buyers, C_shops []int, C_buyers []int){
	file, err := os.OpenFile(DataFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	reader := bufio.NewScanner(file)
	reader.Scan()
	for i := 0; i < Writer_Count; i++ {
		reader.Scan()
		temp := strings.Split(reader.Text(), " ")
		N, err := strconv.Atoi(temp[1])
		if err != nil {
			fmt.Println("Error")
		}
		C_shops= append(C_shops,N)
		for j := 0; j < N; j++{
			reader.Scan()
			temp := strings.Split(reader.Text(), " ")
			memory, err := strconv.Atoi(temp[1])
			if err != nil {
				fmt.Println("Error")
			}
			price, err := strconv.ParseFloat(temp[2], 64)
			if err != nil {
				fmt.Println("Error")
			}//ideda i parduotuviu struktura
			shops.Shop[i].LaptopBox[j].Name=temp[0]
			shops.Shop[i].LaptopBox[j].Memory=memory
			shops.Shop[i].LaptopBox[j].Price=price
		}
	}
	for i := 0; i < Reader_Count; i++ {
		reader.Scan()
		temp := strings.Split(reader.Text(), " ")
		N, err := strconv.Atoi(temp[1])
		if err != nil {
			fmt.Println("Error")
		}
		C_buyers= append(C_buyers,N)
		for j := 0; j < N; j++{
			reader.Scan()
			temp := strings.Split(reader.Text(), " ")
			qnt, err := strconv.Atoi(temp[1])
			if err != nil {
				fmt.Println("Error")
			}//ideda i pirkeju struktura
			buyers.Buyer[i].Inventor[j].Field=temp[0]
			buyers.Buyer[i].Inventor[j].Quantity=qnt
		}
	}
	return
}
//------------Pradiniu duomenu spausdinimas i rezultatu faila
func PrintDataToFile(shops Shops,buyers Buyers,Count_Shops_Items []int, Count_Buyers_Items []int){
file, err := os.Create(RezFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()	
	writer := bufio.NewWriter(file)
	header := fmt.Sprintf("Failas: %1s \r\n", DataFile)				
	writer.WriteString(header)
	writer.WriteString("---------------Duomenys prieš skaitymą---------------------- \r\n")
	writer.WriteString("\r\n")
	for i := 0; i < Writer_Count; i++ {
		header := fmt.Sprintf("---------------Parduotuve %1d---------------- \r\n", i+1)				
		writer.WriteString(header)
		header = fmt.Sprintf("|%3s|%15s|%10s|%10s|\r\n", "Nr","Pavadinimas","Atmintis","Kaina")
		writer.WriteString(header)
		header = fmt.Sprintf("-------------------------------------------\r\n")		
		writer.WriteString(header)
		for j := 0; j < Count_Shops_Items[i]; j++ {
			var name = shops.Shop[i].LaptopBox[j].Name
			var memory = shops.Shop[i].LaptopBox[j].Memory
			var price = shops.Shop[i].LaptopBox[j].Price
			line := fmt.Sprintf("|%3d|%15s|%10d|%10.2f|\r\n",j + 1,name,memory,price)
			writer.WriteString(line)
		}
		writer.WriteString("\r\n")
	}
	for i := 0; i < Reader_Count; i++ {
		header := fmt.Sprintf("--------------Pirkejas %1d-------------- \r\n", i+1)
		writer.WriteString(header)
		header = fmt.Sprintf("|%3s|%16s|%15s|\r\n", "Nr","Pavadinimas","Kiekis")
		writer.WriteString(header)
		header = fmt.Sprintf("--------------------------------------\r\n")		
		writer.WriteString(header)
		for j := 0; j < Count_Buyers_Items[i]; j++ {
			var field = buyers.Buyer[i].Inventor[j].Field
			var quantity = buyers.Buyer[i].Inventor[j].Quantity
			line := fmt.Sprintf("|%3d|%16s|%15d|\r\n",j + 1,field,quantity)
			writer.WriteString(line)
		}
		writer.WriteString("\r\n")
	}
	writer.Flush()
	fmt.Println("PRINTED TO FILE!")
}
//---------------Konteinerio duomenys pridedami i rezultato faila
func AppendToFile(Con ModelContainer, Count int){
	file, err := os.OpenFile(RezFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()	
	writer := bufio.NewWriter(file)
	writer.WriteString("---------------Duomenys po skaitymo--------------------- \r\n")
	header := fmt.Sprintf("---------------Konteineris------------- \r\n")
	count := fmt.Sprintf("Masyvo dydis: %3d \r\n", Count)
	writer.WriteString(header)
	writer.WriteString(count)
	header = fmt.Sprintf("--------------------------------------- \r\n")
	writer.WriteString(header)
	header = fmt.Sprintf("|%3s|%16s|%15s|\r\n", "Nr","Pavadinimas","Kiekis")
	writer.WriteString(header)
	header = fmt.Sprintf("--------------------------------------- \r\n")
	writer.WriteString(header)
	for i := 0; i < Count; i++{
		var field = Con.MBox[i].Field
		var quantity = Con.MBox[i].Quantity
		line := fmt.Sprintf("|%3d|%16s|%15d|\r\n",i + 1,field,quantity)
		writer.WriteString(line)
	}
	writer.Flush()
}