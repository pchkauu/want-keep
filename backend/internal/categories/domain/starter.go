package domain

import household "github.com/pchkauu/want-keep/backend/internal/household/domain"

type StarterDefinition struct {
	ID, ParentID, Key, RU, EN string
}

var starterDefinitions = []StarterDefinition{
	{"24972db1-ed6c-4b19-a307-e99724d65861", "", "housing", "Жильё", "Housing"},
	{"f95d48cf-ddfd-4e7d-881a-fe34e1354be8", "24972db1-ed6c-4b19-a307-e99724d65861", "housing.rent", "Аренда", "Rent"},
	{"4b3b2ccf-3cb5-4761-a89b-d5a5581d070e", "24972db1-ed6c-4b19-a307-e99724d65861", "housing.utilities", "Коммунальные услуги", "Utilities"},
	{"3ec7e6af-303f-4de4-897e-c658d2943b16", "24972db1-ed6c-4b19-a307-e99724d65861", "housing.repairs", "Ремонт", "Repairs"},
	{"38848712-33d0-4bc8-a387-d1fb73d6286d", "24972db1-ed6c-4b19-a307-e99724d65861", "housing.goods", "Товары для дома", "Household goods"},
	{"04a3e865-3149-44b4-a22e-9dd68bc10ac9", "", "food", "Еда", "Food"},
	{"34197cb4-8410-46f8-8447-056d99dad69e", "04a3e865-3149-44b4-a22e-9dd68bc10ac9", "food.groceries", "Домашняя еда и продукты", "Groceries and home food"},
	{"67ff995b-92cd-4474-98fc-b255c71be461", "04a3e865-3149-44b4-a22e-9dd68bc10ac9", "food.restaurants", "Рестораны и кафе", "Restaurants and cafes"},
	{"37e1363b-35cd-4869-998f-6c301044f664", "04a3e865-3149-44b4-a22e-9dd68bc10ac9", "food.delivery", "Доставка готовой еды", "Prepared food delivery"},
	{"a82c6261-76b7-4d45-993d-b6010128e549", "", "transport", "Транспорт", "Transport"},
	{"0e49e90b-1edb-45b7-ab18-e04060d6043e", "a82c6261-76b7-4d45-993d-b6010128e549", "transport.taxi", "Такси", "Taxi"},
	{"4fcfd30e-dea4-4d80-a5c5-914027997550", "a82c6261-76b7-4d45-993d-b6010128e549", "transport.public", "Общественный транспорт", "Public transport"},
	{"ba5241cf-9ba1-402b-a0bf-09005141c405", "a82c6261-76b7-4d45-993d-b6010128e549", "transport.car", "Автомобиль", "Car"},
	{"9f927980-aefe-4d1f-b2b3-2ee5e726a93a", "", "health", "Здоровье", "Health"},
	{"7d0880e3-8991-4205-a3a9-8c12b2b7bbad", "9f927980-aefe-4d1f-b2b3-2ee5e726a93a", "health.doctors", "Врачи", "Doctors"},
	{"b07310ab-c7d7-4b55-9128-a320f357b974", "9f927980-aefe-4d1f-b2b3-2ee5e726a93a", "health.medicine", "Лекарства", "Medicines"},
	{"b3f6377b-d839-4e69-8178-4192d50bd840", "9f927980-aefe-4d1f-b2b3-2ee5e726a93a", "health.insurance", "Страхование", "Insurance"},
	{"6d0d75e5-ac49-48f6-9f32-177c9f8a7181", "", "sport", "Спорт", "Sport"},
	{"c03694f7-c3aa-4e43-a44e-34fdc20b3b35", "6d0d75e5-ac49-48f6-9f32-177c9f8a7181", "sport.gym", "Спортзал", "Gym"},
	{"0945947b-0962-4a15-b093-384bc617e51d", "6d0d75e5-ac49-48f6-9f32-177c9f8a7181", "sport.equipment", "Инвентарь", "Equipment"},
	{"62ee5ff2-319f-4ad5-a4d9-95f37e3b9879", "", "subscriptions", "Подписки и связь", "Subscriptions and communications"},
	{"b40ef66e-ed1f-4809-b8ff-991e59df7343", "62ee5ff2-319f-4ad5-a4d9-95f37e3b9879", "subscriptions.telecom", "Связь и интернет", "Mobile and internet"},
	{"b2588e39-b59b-452c-870c-8c08014c4475", "62ee5ff2-319f-4ad5-a4d9-95f37e3b9879", "subscriptions.digital", "Цифровые подписки", "Digital subscriptions"},
	{"3b7feb8d-d9f1-4139-a813-7819cb1b7bb0", "", "shopping", "Покупки", "Shopping"},
	{"8f15a668-6865-4b92-9611-770873bdd748", "3b7feb8d-d9f1-4139-a813-7819cb1b7bb0", "shopping.clothing", "Одежда", "Clothing"},
	{"a94118b4-9e2c-44c0-8d0d-9a804d092aa0", "3b7feb8d-d9f1-4139-a813-7819cb1b7bb0", "shopping.electronics", "Электроника", "Electronics"},
	{"f9a13a92-2d6d-4ee8-8978-f42cc7f38526", "3b7feb8d-d9f1-4139-a813-7819cb1b7bb0", "shopping.other", "Другие покупки", "Other purchases"},
	{"0565a37b-d547-47cd-92d7-fbd6d23f1ea8", "", "travel", "Путешествия", "Travel"},
	{"65c9b9f7-305f-4c86-9bcb-b89c0fcfa99b", "", "education", "Образование", "Education"},
	{"8c546f11-e201-4465-8b23-e482c3e8be3c", "", "taxes_fees", "Налоги и комиссии", "Taxes and fees"},
	{"2200450b-4566-4bff-a26e-f824a00109de", "", "gifts_charity", "Подарки и благотворительность", "Gifts and charity"},
}

func StarterCategories(family household.HouseholdID) []Category {
	result := make([]Category, 0, len(starterDefinitions))
	for _, definition := range starterDefinitions {
		result = append(result, Category{HouseholdID: family, ID: definition.ID, Revision: 1, ParentID: definition.ParentID, Key: definition.Key, NameRU: definition.RU, NameEN: definition.EN, State: Active, Origin: Starter})
	}
	return result
}
